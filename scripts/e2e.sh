#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
suite=${1:-all}
case "$suite" in
    all|engineer-orders|certificates|director-order|profiles|chat-http|chat-ws) ;;
    *)
        printf 'Usage: %s [all|engineer-orders|certificates|director-order|profiles|chat-http|chat-ws]\n' "$0" >&2
        exit 2
        ;;
esac
api=${API_URL:-http://127.0.0.1/api}
tmp=$(mktemp -d)
inventory_stopped=false
item=10000000-0000-4000-8000-000000000001

cleanup() {
    local result=$?
    if [[ $inventory_stopped == true ]]; then
        docker compose start inventory || result=1
    fi
    rm -r -- "$tmp"
    return "$result"
}
trap cleanup EXIT

request() {
    local expected=$1 path=$2 status
    shift 2
    status=$(curl --silent --show-error --max-time 12 --noproxy '*' --output "$tmp/body" \
        --write-out '%{http_code}' -H 'Host: sto.localhost' "$api$path" "$@")
    if [[ $status != "$expected" ]]; then
        printf '%s returned %s, expected %s\n' "$path" "$status" "$expected" >&2
        cat "$tmp/body" >&2
        return 1
    fi
}

post_json() {
    local status=$1 path=$2 body=$3
    shift 3
    request "$status" "$path" -H 'Content-Type: application/json' --data "$body" "$@"
}

sql() {
    printf '%s\n' "$2" | sql_stream "$1"
}

sql_stream() {
    docker compose exec -T "$1" sh -c \
        'psql -X -q -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -At'
}

register_client() {
    client_phone=$(printf '+79%09d' "$(( (RANDOM * 32768 + RANDOM) % 1000000000 ))")
    post_json 201 /register "$(jq -cn --arg phone "$client_phone" '{name:"Проверка оплаты",phone:$phone,password:"smoke-example-123"}')"
    client_id=$(jq -er '.user_id' "$tmp/body")
    client=$(jq -er '.token' "$tmp/body")
}

prepare_second_engineer() {
    local phone
    phone=$(printf '+78%09d' "$(( (RANDOM * 32768 + RANDOM) % 1000000000 ))")
    second_engineer_id=$(sql users-db "INSERT INTO users (name, phone, password_hash, role, work_started_at)
        SELECT 'Инженер для проверки прав', '$phone', password_hash, 'engineer', CURRENT_DATE
        FROM users WHERE phone = '+70000000001' AND deleted_at IS NULL
        RETURNING id;")
    post_json 200 /login "$(jq -cn --arg phone "$phone" '{phone:$phone,password:"engineer-example"}')"
    second_engineer=$(jq -er '.token' "$tmp/body")
}

prepare_intake_vehicle() {
    local number query
    number="А${client_phone:3}19"
    query=$(jq -rn --arg value "$client_phone" '$value | @uri')
    request 403 "/engineer/clients?q=$query" -H "Authorization: Bearer $client"
    request 200 "/engineer/clients?q=$query" -H "Authorization: Bearer $engineer"
    jq -e --arg id "$client_id" 'any(.[]; .id == $id)' "$tmp/body" >/dev/null

    query=$(jq -rn --arg value "a ${number:1}" '$value | @uri')
    request 404 "/engineer/vehicles?number=$query" -H "Authorization: Bearer $engineer"
    post_json 201 /engineer/vehicles "$(jq -cn --arg owner "$client_id" --arg number "$number" \
        '{owner_id:$owner,number:$number,brand:"Lada",vin:"XTA210990Y2706742"}')" -H "Authorization: Bearer $engineer"
    vehicle_id=$(jq -er '.vehicle_id' "$tmp/body")
    request 200 "/engineer/vehicles?number=$query" -H "Authorization: Bearer $engineer"
    jq -e --arg owner "$client_id" --arg id "$vehicle_id" '.owner_id == $owner and .vehicle_id == $id' "$tmp/body" >/dev/null
    request 200 /vehicles -H "Authorization: Bearer $client"
    jq -e --arg id "$vehicle_id" 'any(.[]; .vehicle_id == $id)' "$tmp/body" >/dev/null
}

create_order() {
    local item_id=${1:-$item}
    post_json 201 /orders "$(jq -cn --arg client "$client_id" --arg vehicle "$vehicle_id" '{client_id:$client,vehicle_id:$vehicle}')" \
        -H "Authorization: Bearer $engineer"
    order_id=$(jq -er '.order_id' "$tmp/body")
    post_json 201 "/orders/$order_id/items" "$(jq -cn --arg item "$item_id" '{item_id:$item,quantity:1}')" \
        -H "Authorization: Bearer $engineer"
}

check_engineer_orders() {
    request 200 "/engineer/orders?scope=active" -H "Authorization: Bearer $engineer"
    jq -e --arg id "$order_id" 'any(.[]; .id == $id and .status == "active" and .vehicle_brand == "Lada" and (.vehicle_number | length > 0))' "$tmp/body" >/dev/null
    request 200 "/engineer/orders?scope=completed" -H "Authorization: Bearer $engineer"
    jq -e --arg id "$previous_order" 'any(.[]; .id == $id and .status == "cancelled")' "$tmp/body" >/dev/null
    request 403 "/orders/$order_id/vehicle-history" -H "Authorization: Bearer $client"
    request 404 "/orders/$previous_order/vehicle-history" -H "Authorization: Bearer $engineer"
    request 200 "/orders/$order_id/vehicle-history" -H "Authorization: Bearer $engineer"
    jq -e --arg current "$order_id" --arg previous "$previous_order" \
        'any(.[]; .id == $previous) and all(.[]; .id != $current)' "$tmp/body" >/dev/null
    check_engineer_access_after_claim
    request 200 "/orders/$order_id" -H "Authorization: Bearer $engineer"
    jq -e '.engineer_id as $responsible | any(.engineers[]; .id == $responsible and (.name | length > 0))' "$tmp/body" >/dev/null
    request 200 "/orders?client_id=me" -H "Authorization: Bearer $client"
    jq -e --arg id "$order_id" 'any(.[]; .id == $id)' "$tmp/body" >/dev/null
}

check_engineer_access_after_claim() {
    request 404 "/orders/$order_id" -H "Authorization: Bearer $second_engineer"
    request 200 "/orders/$order_id/claim" -X PATCH -H "Authorization: Bearer $second_engineer"
    request 200 "/orders/$order_id" -H "Authorization: Bearer $engineer"
    jq -e '.can_edit == false and (.engineers | length == 2)' "$tmp/body" >/dev/null
    request 404 "/orders/$order_id/cancel" -X PATCH -H "Authorization: Bearer $engineer"
    request 404 "/orders/$order_id/vehicle-history/$previous_order" -H "Authorization: Bearer $engineer"
    request 404 "/orders/$previous_order" -H "Authorization: Bearer $second_engineer"
    request 200 "/orders/$order_id/vehicle-history/$previous_order" -H "Authorization: Bearer $second_engineer"
    jq -e --arg id "$previous_order" '.id == $id and .can_edit == false' "$tmp/body" >/dev/null
    request 200 "/orders/$order_id/claim" -X PATCH -H "Authorization: Bearer $engineer"
}

check_intake_retry() {
    local key body original
    key=$(sql orders-db 'SELECT gen_random_uuid();')
    body=$(jq -cn --arg client "$client_id" --arg vehicle "$vehicle_id" --arg key "$key" \
        '{client_id:$client,vehicle_id:$vehicle,request_id:$key}')
    post_json 201 /orders "$body" -H "Authorization: Bearer $engineer"
    original=$(jq -er '.order_id' "$tmp/body")
    post_json 201 /orders "$body" -H "Authorization: Bearer $engineer"
    jq -e --arg id "$original" '.order_id == $id' "$tmp/body" >/dev/null
    body=$(jq -c '.vehicle_id = "00000000-0000-4000-8000-000000000000"' <<< "$body")
    post_json 409 /orders "$body" -H "Authorization: Bearer $engineer"
    jq -e '.error.code == "intake_request_conflict"' "$tmp/body" >/dev/null
}

check_booking_claim() {
    local tomorrow booking_id
    tomorrow=$(date -u -d tomorrow +%F)
    post_json 201 /orders "$(jq -cn --arg vehicle "$vehicle_id" --arg date "$tomorrow" \
        '{vehicle_id:$vehicle,service:"Проверка заявки",date:$date}')" -H "Authorization: Bearer $client"
    booking_id=$(jq -er '.order_id' "$tmp/body")
    request 200 "/orders?status=requested" -H "Authorization: Bearer $engineer"
    jq -e --arg id "$booking_id" 'any(.[]; .id == $id)' "$tmp/body" >/dev/null

    request 200 "/orders/$booking_id/claim" -X PATCH -H "Authorization: Bearer $engineer"
    request 200 "/orders?status=requested" -H "Authorization: Bearer $engineer"
    jq -e --arg id "$booking_id" 'all(.[]; .id != $id)' "$tmp/body" >/dev/null
    request 200 "/engineer/orders?scope=active" -H "Authorization: Bearer $engineer"
    jq -e --arg id "$booking_id" 'any(.[]; .id == $id and .status == "active")' "$tmp/body" >/dev/null
    printf 'Booking moved from the pool to assigned orders: %s\n' "$booking_id"
}

pay() {
    request 200 "/orders/$order_id/pay" -X POST -H "Authorization: Bearer $client"
    jq -e '.status == "paid" and (.paid_at | type == "string")' "$tmp/body" >/dev/null
}

wait_for_order() {
    local filter=$1
    for ((attempt=0; attempt<30; attempt++)); do
        request 200 "/orders/$order_id" -H "Authorization: Bearer $client"
        if jq -e "$filter" "$tmp/body" >/dev/null; then
            return 0
        fi
        sleep 1
    done
    printf 'Order %s did not match %s\n' "$order_id" "$filter" >&2
    return 1
}

stock() {
    sql inventory-db "SELECT quantity FROM inventory WHERE id = '$item'"
}

wait_for_pdf() {
    local status
    for ((attempt=0; attempt<60; attempt++)); do
        status=$(curl --silent --show-error --max-time 10 --noproxy '*' --output "$tmp/certificate.pdf" \
            --write-out '%{http_code}' -H 'Host: sto.localhost' -H "Authorization: Bearer $client" "$api/orders/$order_id/certificate")
        if [[ $status == 200 ]]; then
            [[ $(head -c 5 "$tmp/certificate.pdf") == '%PDF-' ]]
            if command -v pdftotext >/dev/null; then
                pdftotext "$tmp/certificate.pdf" "$tmp/certificate.txt"
                rg -q 'Справка об оплате' "$tmp/certificate.txt"
                rg -q 'Масляный фильтр' "$tmp/certificate.txt"
            fi
            return
        fi
        [[ $status == 404 ]]
        sleep 1
    done
    printf 'Certificate did not become ready for %s\n' "$order_id" >&2
    return 1
}

check_duplicate() {
    local saga_id=$1 expected_stock=$2
    sql orders-db "UPDATE outbox SET published = false WHERE aggregate_id = '$saga_id' AND event_type = 'DeductStock'" >/dev/null
    for ((attempt=0; attempt<20; attempt++)); do
        if [[ $(sql inventory-db "SELECT count(*) FROM outbox WHERE aggregate_id = '$saga_id' AND event_type = 'StockDeducted' AND published") == 2 ]]; then
            [[ $(stock) == "$expected_stock" ]]
            return
        fi
        sleep 1
    done
    printf 'Duplicate deduction did not produce a repeated reply for %s\n' "$saga_id" >&2
    return 1
}

check_payment_denials() {
    request 403 "/orders/$order_id/pay" -X POST -H "Authorization: Bearer $engineer"
    request 409 "/orders/$order_id/pay" -X POST -H "Authorization: Bearer $other_client"
    request 200 "/orders/$order_id" -H "Authorization: Bearer $client"
    jq -e '.status == "active" and (.payment_confirmed | not)' "$tmp/body" >/dev/null

    request 200 "/orders/$order_id/cancel" -X PATCH -H "Authorization: Bearer $engineer"
    request 409 "/orders/$order_id/pay" -X POST -H "Authorization: Bearer $client"
    request 200 "/orders/$order_id" -H "Authorization: Bearer $client"
    jq -e '.status == "cancelled" and (.payment_confirmed | not)' "$tmp/body" >/dev/null
    [[ $(sql orders-db "SELECT count(*) FROM saga_state WHERE order_id = '$order_id'") == 0 ]]
}

success_case() {
    local before saga_id paid_at
    before=$(stock)
    create_order
    pay
    paid_at=$(jq -er '.paid_at' "$tmp/body")
    pay
    [[ $(jq -er '.paid_at' "$tmp/body") == "$paid_at" ]]
    wait_for_order '.status == "paid" and .payment_confirmed and .stock_status == "deducted"'
    [[ $(stock) == "$((before - 1))" ]]
    saga_id=$(sql orders-db "SELECT saga_id FROM saga_state WHERE order_id = '$order_id' AND current_step = 'completed'")
    [[ -n $saga_id ]]
    check_duplicate "$saga_id" "$((before - 1))"
    wait_for_pdf
    printf 'Payment confirmed, duplicate handled: %s\n' "$order_id"
}

check_handover() {
    post_json 200 "/orders/$order_id/stage" '{"stage":"выдан"}' -X PATCH -H "Authorization: Bearer $engineer"
    request 200 "/engineer/orders?scope=active" -H "Authorization: Bearer $engineer"
    jq -e --arg id "$order_id" 'all(.[]; .id != $id)' "$tmp/body" >/dev/null
    request 200 "/engineer/orders?scope=completed" -H "Authorization: Bearer $engineer"
    jq -e --arg id "$order_id" 'any(.[]; .id == $id and .stage == "выдан")' "$tmp/body" >/dev/null
    request 404 "/orders/$order_id/vehicle-history" -H "Authorization: Bearer $engineer"
}

check_engineer_order_workflow() {
    create_order
    request 200 "/orders/$order_id/cancel" -X PATCH -H "Authorization: Bearer $engineer"
    previous_order=$order_id
    create_order
    check_engineer_orders
    pay
    wait_for_order '.status == "paid" and .payment_confirmed and .stock_status == "deducted"'
    check_handover
    wait_for_pdf
    printf 'Engineer order, history access, handover and PDF checked: %s\n' "$order_id"
}

unavailable_inventory_case() {
    local before
    before=$(stock)
    create_order
    inventory_stopped=true
    docker compose stop inventory
    pay
    sleep 11
    wait_for_order '.status == "paid" and .payment_confirmed and .stock_status == "pending"'
    request 404 "/orders/$order_id/certificate" -H "Authorization: Bearer $client"
    post_json 409 "/orders/$order_id/stage" '{"stage":"выдан"}' -X PATCH -H "Authorization: Bearer $engineer"
    [[ $(stock) == "$before" ]]
    docker compose start inventory
    inventory_stopped=false
    wait_for_order '.status == "paid" and .payment_confirmed and .stock_status == "deducted"'
    [[ $(stock) == "$((before - 1))" ]]
    check_handover
    wait_for_pdf
    printf 'Payment stayed confirmed while Inventory was stopped: %s\n' "$order_id"
}

rejected_stock_case() {
    local test_item
    test_item=$(sql inventory-db "WITH item AS (INSERT INTO inventory (name, price, quantity) VALUES ('Масляный фильтр — проверка списания', 1, 0) RETURNING id) SELECT id FROM item")
    create_order "$test_item"
    pay
    wait_for_order '.status == "paid" and .payment_confirmed and .stock_status == "rejected"'
    request 403 "/orders/$order_id/stock/retry" -X POST -H "Authorization: Bearer $client"
    request 409 "/orders/$order_id/stock/retry" -X POST -H "Authorization: Bearer $second_engineer"
    sql inventory-db "UPDATE inventory SET quantity = 1 WHERE id = '$test_item'" >/dev/null
    request 202 "/orders/$order_id/stock/retry" -X POST -H "Authorization: Bearer $engineer"
    request 409 "/orders/$order_id/stock/retry" -X POST -H "Authorization: Bearer $engineer"
    wait_for_order '.status == "paid" and .payment_confirmed and .stock_status == "deducted"'
    wait_for_pdf
    printf 'Rejected stock deduction succeeded after replenishment: %s\n' "$order_id"
}

create_certificate_fixture() {
    local attempts=$1 next_attempt=$2
    # У фикстуры запланирован повтор, но воркер её не подберёт.
    sql orders-db "WITH created AS (
        INSERT INTO orders (client_id, vehicle_id, status, paid_at)
        VALUES ('$client_id', '$vehicle_id', 'paid', now()) RETURNING id
    )
    INSERT INTO certificates (order_id, status, attempts, next_attempt_at)
    SELECT id, 'failed', $attempts, $next_attempt FROM created
    RETURNING order_id;"
}

check_client_certificates() {
    local ready_order processing_order failed_order
    register_client
    other_client=$client
    register_client
    create_client_vehicle
    create_order
    pay
    wait_for_pdf
    ready_order=$order_id
    processing_order=$(create_certificate_fixture 1 "'infinity'::timestamptz")
    failed_order=$(create_certificate_fixture 3 NULL)

    request 200 /certificates -H "Authorization: Bearer $client"
    jq -e --arg ready "$ready_order" --arg processing "$processing_order" --arg failed "$failed_order" \
        'length == 3 and any(.[]; .order_id == $ready and .status == "ready")
            and any(.[]; .order_id == $processing and .status == "processing")
            and any(.[]; .order_id == $failed and .status == "failed")' "$tmp/body" >/dev/null
    request 200 /certificates -H "Authorization: Bearer $other_client"
    jq -e '. == []' "$tmp/body" >/dev/null
    request 404 "/orders/$ready_order/certificate" -H "Authorization: Bearer $other_client"
    request 403 /certificates -H "Authorization: Bearer $engineer"
    request 200 "/orders/$ready_order" -H "Authorization: Bearer $engineer"
    jq -e '.certificate_status == "ready"' "$tmp/body" >/dev/null
    request 403 "/orders/$ready_order/certificate" -H "Authorization: Bearer $engineer"
    login_director
    request 200 "/director/orders/$ready_order" -H "Authorization: Bearer $director"
    jq -e '.order.certificate_status == "ready"' "$tmp/body" >/dev/null
    request 200 "/orders/$ready_order/certificate" -H "Authorization: Bearer $director"
    [[ $(head -c 5 "$tmp/body") == '%PDF-' ]]
    printf 'Certificate statuses and client, engineer and director permissions checked\n'
}

create_client_vehicle() {
    post_json 201 /vehicles "$(jq -cn --arg number "С${client_phone:3}19" \
        '{number:$number,brand:"Lada",vin:"XTA210990Y2706742"}')" -H "Authorization: Bearer $client"
    vehicle_id=$(jq -er '.vehicle_id' "$tmp/body")
}

login_director() {
    post_json 200 /login '{"phone":"+70000000002","password":"director-example"}'
    director=$(jq -er '.token' "$tmp/body")
}

check_director_order() {
    local item_id service_id
    register_client
    create_client_vehicle
    login_director
    post_json 201 /inventory '{"name":"Товар для проверки цены","quantity":5,"price":"123.45"}' -H "Authorization: Bearer $director"
    item_id=$(jq -er '.id' "$tmp/body")
    post_json 201 /services '{"name":"Услуга для проверки цены","price":"456.78"}' -H "Authorization: Bearer $director"
    service_id=$(jq -er '.id' "$tmp/body")
    create_order "$item_id"
    post_json 201 "/orders/$order_id/services" "$(jq -cn --arg service "$service_id" \
        '{service_id:$service,quantity:1}')" -H "Authorization: Bearer $engineer"

    post_json 200 "/inventory/$item_id" '{"name":"Товар с новой ценой","quantity":5,"price":"999.00"}' \
        -X PUT -H "Authorization: Bearer $director"
    post_json 200 "/services/$service_id" '{"name":"Услуга с новой ценой","price":"999.00"}' \
        -X PUT -H "Authorization: Bearer $director"
    request 403 "/director/orders/$order_id" -H "Authorization: Bearer $client"
    request 403 "/director/orders/$order_id" -H "Authorization: Bearer $engineer"
    request 200 "/director/orders/$order_id" -H "Authorization: Bearer $director"
    jq -e --arg id "$order_id" --arg phone "$client_phone" \
        '.order.id == $id and .client.phone == $phone and .vehicle.brand == "Lada"
        and .vehicle.vin == "XTA210990Y2706742" and (.order.engineers | length == 1)
        and .order.status == "active" and .order.stage == "принят"
        and (.order.items[0].price | tonumber) == 123.45
        and (.order.services[0].price | tonumber) == 456.78
        and (.order.total | tonumber) == 580.23' "$tmp/body" >/dev/null
    printf 'Director details preserve saved prices: %s\n' "$order_id"
}

check_profiles() {
    local profile_password='smoke-example-123'
    request 401 /me
    register_client
    request 200 /me -H "Authorization: Bearer $client"
    jq -e --arg phone "$client_phone" '.role == "client" and .phone == $phone and .notifications_enabled == false and (.created_at | type == "string")' "$tmp/body" >/dev/null
    post_json 200 /me '{"notifications_enabled":true}' -X PATCH -H "Authorization: Bearer $client"
    jq -e '.notifications_enabled == true' "$tmp/body" >/dev/null
    post_json 200 /login "$(jq -cn --arg phone "$client_phone" --arg password "$profile_password" \
        '{phone:$phone,password:$password}')"
    client=$(jq -er '.token' "$tmp/body")
    request 200 /me -H "Authorization: Bearer $client"
    jq -e '.notifications_enabled == true' "$tmp/body" >/dev/null
    post_json 200 /me '{"notifications_enabled":false}' -X PATCH -H "Authorization: Bearer $client"
    jq -e '.notifications_enabled == false' "$tmp/body" >/dev/null
    request 200 /me -H "Authorization: Bearer $engineer"
    jq -e '.role == "engineer" and (.work_started_at | type == "string")' "$tmp/body" >/dev/null
    login_director
    request 200 /me -H "Authorization: Bearer $director"
    jq -e '.role == "director" and (.work_started_at | type == "string")' "$tmp/body" >/dev/null
    printf 'Profiles and notification setting persistence checked\n'
}

check_chat_http() {
    request 401 /chat/dialogs
    login_director
    prepare_second_engineer
    register_client
    other_client=$client
    register_client
    create_client_vehicle
    check_order_chat
    check_support_and_staff_chat
    printf 'Chat HTTP checked for all three dialog types\n'
}

check_chat_websocket() {
    request 401 /chat/events
    request 401 /chat/events -H 'Sec-WebSocket-Protocol: sto.chat.v1, invalid'
    prepare_second_engineer
    register_client
    other_client=$client
    register_client
    create_client_vehicle

    API_URL="$api" CHAT_CLIENT="$client" CHAT_OUTSIDER="$other_client" CHAT_ENGINEER="$engineer" \
        CHAT_SECOND_ENGINEER="$second_engineer" CHAT_VEHICLE_ID="$vehicle_id" node scripts/e2e-chat-ws.mjs
}

check_order_chat() {
    local tomorrow booking open dialog messages first read_body
    tomorrow=$(date -u -d tomorrow +%F)
    post_json 201 /orders "$(jq -cn --arg vehicle "$vehicle_id" --arg date "$tomorrow" \
        '{vehicle_id:$vehicle,service:"Проверка чата",date:$date}')" -H "Authorization: Bearer $client"
    booking=$(jq -er '.order_id' "$tmp/body")
    open=$(jq -cn --arg id "$booking" '{order_id:$id}')
    request 403 "/internal/orders/$booking/chat-participants" -H "Authorization: Bearer $client"
    post_json 409 /chat/dialogs/order "$open" -H "Authorization: Bearer $client"
    jq -e '.error.code == "order_engineer_not_assigned"' "$tmp/body" >/dev/null
    request 200 "/orders/$booking/claim" -X PATCH -H "Authorization: Bearer $engineer"
    post_json 200 /chat/dialogs/order "$open" -H "Authorization: Bearer $client"
    dialog=$(jq -er '.dialog_id' "$tmp/body")
    messages="/chat/dialogs/$dialog/messages"
    post_json 200 /chat/dialogs/order "$open" -H "Authorization: Bearer $client"
    jq -e --arg dialog "$dialog" '.dialog_id == $dialog' "$tmp/body" >/dev/null
    post_json 404 /chat/dialogs/order "$open" -H "Authorization: Bearer $other_client"
    request 404 "$messages" -H "Authorization: Bearer $other_client"
    request 200 /chat/dialogs -H "Authorization: Bearer $other_client"
    jq -e --arg dialog "$dialog" 'all(.[]; .id != $dialog)' "$tmp/body" >/dev/null

    post_json 201 "$messages" '{"body":"первое"}' -H "Authorization: Bearer $client"
    first=$(jq -er 'select(.is_read == false).id' "$tmp/body")
    post_json 201 "$messages" '{"body":"второе"}' -H "Authorization: Bearer $client"
    read_body=$(jq -cn --arg id "$first" '{through_message_id:$id}')
    post_json 204 "/chat/dialogs/$dialog/read" "$read_body" -X PATCH -H "Authorization: Bearer $engineer"
    post_json 204 "/chat/dialogs/$dialog/read" "$read_body" -X PATCH -H "Authorization: Bearer $engineer"
    request 200 "$messages" -H "Authorization: Bearer $client"
    jq -e 'map(.is_read) == [true,false]' "$tmp/body" >/dev/null
    request 200 /chat/dialogs -H "Authorization: Bearer $engineer"
    jq -e --arg dialog "$dialog" 'any(.[]; .id == $dialog and .can_write and .unread_count == 1)' "$tmp/body" >/dev/null

    request 200 "/orders/$booking/claim" -X PATCH -H "Authorization: Bearer $second_engineer"
    request 404 "$messages" -H "Authorization: Bearer $engineer"
    post_json 404 "$messages" '{"body":"поздний ответ"}' -H "Authorization: Bearer $engineer"
    request 200 /chat/dialogs -H "Authorization: Bearer $engineer"
    jq -e --arg dialog "$dialog" 'all(.[]; .id != $dialog)' "$tmp/body" >/dev/null
    request 200 "$messages" -H "Authorization: Bearer $client"
    jq -e 'length == 2' "$tmp/body" >/dev/null
    post_json 403 "$messages" '{"body":"архив"}' -H "Authorization: Bearer $client"
    jq -e '.error.code == "dialog_read_only"' "$tmp/body" >/dev/null
    post_json 403 "/chat/dialogs/$dialog/read" "$read_body" -X PATCH -H "Authorization: Bearer $client"
    request 200 /chat/dialogs -H "Authorization: Bearer $client"
    jq -e --arg dialog "$dialog" 'any(.[]; .id == $dialog and (.can_write | not))' "$tmp/body" >/dev/null

    post_json 200 /chat/dialogs/order "$open" -H "Authorization: Bearer $client"
    jq -e --arg old "$dialog" '.dialog_id != $old' "$tmp/body" >/dev/null
    dialog=$(jq -er '.dialog_id' "$tmp/body")
    request 200 "/chat/dialogs/$dialog/messages" -H "Authorization: Bearer $second_engineer"
    jq -e 'length == 0' "$tmp/body" >/dev/null
}

check_support_and_staff_chat() {
    local support staff body
    request 200 /chat/dialogs/support -X POST -H "Authorization: Bearer $client"
    support=$(jq -er '.dialog_id' "$tmp/body")
    request 200 /chat/dialogs/support -X POST -H "Authorization: Bearer $client"
    jq -e --arg id "$support" '.dialog_id == $id' "$tmp/body" >/dev/null
    post_json 201 "/chat/dialogs/$support/messages" '{"body":"нужна помощь"}' -H "Authorization: Bearer $client"
    request 200 "/chat/dialogs/$support/messages" -H "Authorization: Bearer $director"
    jq -e 'any(.[]; .body == "нужна помощь")' "$tmp/body" >/dev/null
    request 404 "/chat/dialogs/$support/messages" -H "Authorization: Bearer $engineer"

    body=$(jq -cn --arg id "$second_engineer_id" '{engineer_id:$id}')
    post_json 403 /chat/dialogs/director-engineer "$body" -H "Authorization: Bearer $client"
    post_json 200 /chat/dialogs/director-engineer "$body" -H "Authorization: Bearer $director"
    staff=$(jq -er '.dialog_id' "$tmp/body")
    post_json 200 /chat/dialogs/director-engineer "$body" -H "Authorization: Bearer $director"
    jq -e --arg id "$staff" '.dialog_id == $id' "$tmp/body" >/dev/null
    post_json 201 "/chat/dialogs/$staff/messages" '{"body":"сообщение директору"}' -H "Authorization: Bearer $second_engineer"
    request 200 "/chat/dialogs/$staff/messages" -H "Authorization: Bearer $director"
    jq -e 'any(.[]; .body == "сообщение директору")' "$tmp/body" >/dev/null
    request 404 "/chat/dialogs/$staff/messages" -H "Authorization: Bearer $client"
}

check_v1_payment_backfill() {
    # Временные таблицы повторяют нужные поля схемы v1 и скрывают рабочие таблицы
    # в этом сеансе psql. Применяем тот же SQL, который выполняет goose при переходе.
    local result
    result=$({
        cat <<'SQL'
CREATE TEMP TABLE orders (id int PRIMARY KEY, status text NOT NULL);
CREATE TEMP TABLE saga_state (order_id int, current_step text, updated_at timestamptz);
INSERT INTO orders VALUES (1, 'paid'), (2, 'paid'), (3, 'active');
INSERT INTO saga_state VALUES
    (1, 'failed', '2026-11-23 13:59+03'),
    (1, 'completed', '2026-11-23 14:00+03'),
    (2, 'verifying_stock', '2026-11-24 14:00+03'),
    (3, 'failed', '2026-11-24 14:00+03');
SQL
        cat orders/migrations/00007_order_paid_at.sql
        cat <<'SQL'
SELECT count(*) = 3 AND count(paid_at) = 1
    AND bool_or(id = 1 AND paid_at = '2026-11-23 14:00+03'::timestamptz)
FROM orders;
SQL
    } | sql_stream orders-db)
    [[ $result == t ]]
    printf 'Migration preserved the v1 payment timestamp: 00007\n'
}

create_v1_attempt() {
    local saga_id=$1 deadline=$2
    sql orders-db "BEGIN;
        UPDATE orders SET status = 'paid' WHERE id = '$order_id' AND status = 'active';
        INSERT INTO saga_state (saga_id, order_id, current_step, deadline)
        VALUES ('$saga_id', '$order_id', 'verifying_stock', clock_timestamp() + interval '$deadline');
        COMMIT;"
}

v1_success_case() {
    local before saga_id paid_at
    before=$(stock)
    create_order
    saga_id=$(sql orders-db 'SELECT gen_random_uuid();')
    create_v1_attempt "$saga_id" '2 minutes'
    sql orders-db "INSERT INTO outbox (aggregate_id, event_type, payload)
        VALUES ('$saga_id', 'DeductStock', jsonb_build_object('saga_id', '$saga_id', 'order_id', '$order_id',
            'items', jsonb_build_array(jsonb_build_object('item_id', '$item', 'qty', 1))));"

    wait_for_order '.status == "paid" and .payment_confirmed and .stock_status == "deducted"'
    [[ $(sql orders-db "SELECT o.paid_at = s.updated_at FROM orders o
        JOIN saga_state s ON s.order_id = o.id WHERE s.saga_id = '$saga_id'") == t ]]
    wait_for_pdf
    cp "$tmp/certificate.pdf" "$tmp/legacy-certificate.pdf"

    pay
    paid_at=$(jq -er '.paid_at' "$tmp/body")
    check_duplicate "$saga_id" "$((before - 1))"
    pay
    [[ $(jq -er '.paid_at' "$tmp/body") == "$paid_at" ]]
    [[ $(sql orders-db "SELECT count(*) FROM saga_state WHERE order_id = '$order_id'") == 1 ]]
    wait_for_pdf
    cmp -s "$tmp/legacy-certificate.pdf" "$tmp/certificate.pdf"
    printf 'In-flight v1 payment completed and repeat preserved the PDF: %s\n' "$order_id"
}

v1_compensation_case() {
    local saga_id test_item
    test_item=$(sql inventory-db "INSERT INTO inventory (name, price, quantity)
        VALUES ('Масляный фильтр — проверка компенсации', 1, 1) RETURNING id")
    create_order "$test_item"
    saga_id=$(sql orders-db 'SELECT gen_random_uuid();')
    # Склад уже списал товар v1, но Orders ещё не получил ответ к дедлайну.
    sql inventory-db "BEGIN;
        UPDATE inventory SET quantity = 0 WHERE id = '$test_item';
        INSERT INTO stock_attempts (saga_id, order_id, state) VALUES ('$saga_id', '$order_id', 'deducted');
        INSERT INTO stock_attempt_items (saga_id, item_id, quantity) VALUES ('$saga_id', '$test_item', 1);
        COMMIT;"
    create_v1_attempt "$saga_id" '-1 second'
    wait_for_order '.status == "active" and (.payment_confirmed | not)'
    request 404 "/orders/$order_id/certificate" -H "Authorization: Bearer $client"

    for ((attempt=0; attempt<25; attempt++)); do
        if [[ $(sql inventory-db "SELECT state FROM stock_attempts WHERE saga_id = '$saga_id'") == cancelled ]]; then
            [[ $(sql inventory-db "SELECT quantity FROM inventory WHERE id = '$test_item'") == 1 ]]
            printf 'Expired v1 attempt restored the order and stock: %s\n' "$order_id"
            return
        fi
        sleep 1
    done
    printf 'Cancellation did not reach Inventory for %s\n' "$saga_id" >&2
    return 1
}

post_json 200 /login '{"phone":"+70000000001","password":"engineer-example"}'
engineer=$(jq -er '.token' "$tmp/body")
case "$suite" in
    certificates) check_client_certificates; exit ;;
    director-order) check_director_order; exit ;;
    profiles) check_profiles; exit ;;
    chat-http) check_chat_http; exit ;;
    chat-ws) check_chat_websocket; exit ;;
esac
prepare_second_engineer
if [[ $suite == all ]]; then
    register_client
    other_client=$client
fi
register_client
prepare_intake_vehicle
check_intake_retry
check_booking_claim
check_engineer_order_workflow
rejected_stock_case
if [[ $suite == all ]]; then
    create_order
    check_payment_denials
    success_case
    unavailable_inventory_case
    check_v1_payment_backfill
    v1_success_case
    v1_compensation_case
    check_client_certificates
    check_director_order
    check_profiles
fi
