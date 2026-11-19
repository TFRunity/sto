-- name: CreateOrder :one
INSERT INTO orders (client_id, vehicle_id, engineer_id, status)
VALUES (sqlc.arg(client_id), sqlc.arg(vehicle_id), sqlc.arg(engineer_id)::uuid, 'active')
RETURNING id;

-- name: AddOrderEngineer :exec
INSERT INTO order_engineers (order_id, engineer_id)
VALUES ($1, $2)
ON CONFLICT (order_id, engineer_id) DO NOTHING;

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
