#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
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
    local service=$1 query=$2
    docker compose exec -T "$service" sh -c \
        'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "$1"' sh "$query"
}

register_client() {
    local phone
    phone=$(printf '+79%09d' "$(( (RANDOM * 32768 + RANDOM) % 1000000000 ))")
    post_json 201 /register "$(jq -cn --arg phone "$phone" '{name:"Проверка оплаты",phone:$phone,password:"smoke-example-123"}')"
    client_id=$(jq -er '.user_id' "$tmp/body")
    client=$(jq -er '.token' "$tmp/body")
}

create_order() {
    local item_id=${1:-$item}
    post_json 201 /orders "$(jq -cn --arg client "$client_id" '{client_id:$client,vehicle_id:"20000000-0000-4000-8000-000000000002"}')" \
        -H "Authorization: Bearer $engineer"
    order_id=$(jq -er '.order_id' "$tmp/body")
    post_json 201 "/orders/$order_id/items" "$(jq -cn --arg item "$item_id" '{item_id:$item,quantity:1}')" \
        -H "Authorization: Bearer $engineer"
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

success_case() {
    local before saga_id paid_at
    before=$(stock)
    create_order
    request 403 "/orders/$order_id/pay" -X POST -H "Authorization: Bearer $engineer"
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
    post_json 200 "/orders/$order_id/stage" '{"stage":"выдан"}' -X PATCH -H "Authorization: Bearer $engineer"
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
    sql inventory-db "UPDATE inventory SET quantity = 1 WHERE id = '$test_item'" >/dev/null
    request 202 "/orders/$order_id/stock/retry" -X POST -H "Authorization: Bearer $engineer"
    request 409 "/orders/$order_id/stock/retry" -X POST -H "Authorization: Bearer $engineer"
    wait_for_order '.status == "paid" and .payment_confirmed and .stock_status == "deducted"'
    wait_for_pdf
    printf 'Rejected stock deduction succeeded after replenishment: %s\n' "$order_id"
}

post_json 200 /login '{"phone":"+70000000001","password":"engineer-example"}'
engineer=$(jq -er '.token' "$tmp/body")
register_client
success_case
unavailable_inventory_case
rejected_stock_case
