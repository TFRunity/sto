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
    post_json 201 /orders "$(jq -cn --arg client "$client_id" '{client_id:$client,vehicle_id:"20000000-0000-4000-8000-000000000002"}')" \
        -H "Authorization: Bearer $engineer"
    order_id=$(jq -er '.order_id' "$tmp/body")
    post_json 201 "/orders/$order_id/items" "$(jq -cn --arg item "$item" '{item_id:$item,quantity:1}')" \
        -H "Authorization: Bearer $engineer"
}

pay() {
    request 202 "/orders/$order_id/pay" -X POST -H "Authorization: Bearer $client"
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
    local before saga_id
    before=$(stock)
    create_order
    request 403 "/orders/$order_id/pay" -X POST -H "Authorization: Bearer $engineer"
    pay
    wait_for_order '.status == "paid" and .payment_confirmed == true'
    [[ $(stock) == "$((before - 1))" ]]
    saga_id=$(sql orders-db "SELECT saga_id FROM saga_state WHERE order_id = '$order_id' AND current_step = 'completed'")
    [[ -n $saga_id ]]
    check_duplicate "$saga_id" "$((before - 1))"
    wait_for_pdf
    printf 'Payment confirmed, duplicate handled: %s\n' "$order_id"
}

deadline_case() {
    local before saga_id
    before=$(stock)
    create_order
    inventory_stopped=true
    docker compose stop inventory
    pay
    wait_for_order '.status == "active" and .payment_confirmed == false'
    saga_id=$(sql orders-db "SELECT saga_id FROM saga_state WHERE order_id = '$order_id' AND current_step = 'failed'")
    [[ -n $saga_id ]]
    docker compose start inventory
    inventory_stopped=false
    for ((attempt=0; attempt<25; attempt++)); do
        if [[ $(sql inventory-db "SELECT state FROM stock_attempts WHERE saga_id = '$saga_id'") == cancelled ]]; then
            [[ $(stock) == "$before" ]]
            printf 'Deadline restored order and stock: %s\n' "$order_id"
            return
        fi
        sleep 1
    done
    printf 'Cancellation did not reach Inventory for %s\n' "$saga_id" >&2
    return 1
}

post_json 200 /login '{"phone":"+70000000001","password":"engineer-example"}'
engineer=$(jq -er '.token' "$tmp/body")
register_client
success_case
deadline_case
