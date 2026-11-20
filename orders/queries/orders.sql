-- name: CreateOrder :one
INSERT INTO orders (client_id, vehicle_id, engineer_id, status)
VALUES (sqlc.arg(client_id), sqlc.arg(vehicle_id), sqlc.arg(engineer_id)::uuid, 'active')
RETURNING id;

-- name: CreateServiceBooking :one
INSERT INTO orders (client_id, vehicle_id, status, requested_service, desired_date, comment)
VALUES (sqlc.arg(client_id), sqlc.arg(vehicle_id), 'requested', sqlc.arg(requested_service), sqlc.arg(desired_date), sqlc.narg(comment))
RETURNING id;

-- name: AddOrderEngineer :exec
INSERT INTO order_engineers (order_id, engineer_id)
VALUES ($1, $2)
ON CONFLICT (order_id, engineer_id) DO NOTHING;

-- OpenAPI проверяет status=requested, SQL закрепляет этот фильтр.
-- id задаёт порядок для заявок с одинаковым created_at.

-- name: ListServiceBookings :many
SELECT id, client_id, vehicle_id, requested_service, desired_date, comment, created_at
FROM orders
WHERE status = 'requested'
ORDER BY created_at, id;

-- name: AddOrderItem :one
INSERT INTO order_items (order_id, item_id, quantity, price)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: GetOrder :one
SELECT o.id, o.client_id, o.vehicle_id, o.engineer_id, o.status, o.stage, o.created_at,
       EXISTS (
           SELECT 1
           FROM saga_state AS s
           WHERE s.order_id = o.id AND s.current_step = 'completed'
       ) AS payment_confirmed
FROM orders AS o
WHERE o.id = $1;

-- name: ListOrderItems :many
SELECT id, item_id, quantity, price
FROM order_items
WHERE order_id = $1
ORDER BY created_at, id;
