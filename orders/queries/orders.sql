-- name: CreateOrder :one
INSERT INTO orders (client_id, vehicle_id, engineer_id, status, intake_engineer_id, intake_request_id)
VALUES (sqlc.arg(client_id), sqlc.arg(vehicle_id), sqlc.arg(engineer_id)::uuid, 'active',
        CASE WHEN sqlc.narg(request_id)::uuid IS NOT NULL THEN sqlc.arg(engineer_id)::uuid END,
        sqlc.narg(request_id)::uuid)
ON CONFLICT ON CONSTRAINT uq_orders_intake_request DO NOTHING
RETURNING id;

-- name: GetIntakeOrder :one
SELECT id, client_id, vehicle_id
FROM orders
WHERE intake_engineer_id = sqlc.arg(engineer_id)::uuid
  AND intake_request_id = sqlc.arg(request_id)::uuid;

-- name: CreateServiceBooking :one
INSERT INTO orders (client_id, vehicle_id, status, requested_service, desired_date, comment)
VALUES (sqlc.arg(client_id), sqlc.arg(vehicle_id), 'requested', sqlc.arg(requested_service), sqlc.arg(desired_date), sqlc.narg(comment))
RETURNING id;

-- name: AddOrderEngineer :exec
INSERT INTO order_engineers (order_id, engineer_id)
VALUES ($1, $2)
ON CONFLICT (order_id, engineer_id) DO NOTHING;

-- По ADR-017 повторное назначение active допустимо. Оплаченные и
-- отменённые заказы остаются в своих статусах.

-- name: ClaimOrder :execrows
UPDATE orders
SET status = 'active', engineer_id = sqlc.arg(engineer_id)::uuid, updated_at = now()
WHERE id = sqlc.arg(order_id) AND status IN ('requested', 'active');

-- OpenAPI проверяет status=requested, SQL закрепляет этот фильтр.
-- id задаёт порядок для заявок с одинаковым created_at.

-- name: ListServiceBookings :many
SELECT id, client_id, vehicle_id, requested_service, desired_date, comment, created_at
FROM orders
WHERE status = 'requested'
ORDER BY created_at, id;

-- name: AddOrderItem :one
INSERT INTO order_items (order_id, item_id, name, quantity, price)
SELECT o.id, sqlc.arg(item_id), sqlc.narg(name), sqlc.arg(quantity), sqlc.arg(price)
FROM orders AS o
WHERE o.id = sqlc.arg(order_id) AND o.engineer_id = sqlc.arg(engineer_id)::uuid AND o.status = 'active'
FOR UPDATE OF o
RETURNING id;

-- name: ListClientOrderHistory :many
SELECT o.id, o.vehicle_id, o.status, o.stage, o.created_at,
       (o.paid_at IS NOT NULL OR EXISTS (
           SELECT 1
           FROM saga_state AS s
           WHERE s.order_id = o.id AND s.current_step = 'completed'
       )) AS payment_confirmed
FROM orders AS o
WHERE o.client_id = sqlc.arg(client_id)
  AND (sqlc.narg(vehicle_id)::uuid IS NULL OR o.vehicle_id = sqlc.narg(vehicle_id)::uuid)
ORDER BY o.created_at DESC, o.id DESC;

-- name: GetOrder :one
SELECT o.id, o.client_id, o.vehicle_id, o.engineer_id, o.status, o.stage, o.created_at,
       (o.paid_at IS NOT NULL OR stock.status = 'deducted') AS payment_confirmed,
       stock.status::text AS stock_status
FROM orders AS o
CROSS JOIN LATERAL (
    SELECT CASE
        WHEN count(*) FILTER (WHERE current_step = 'completed') > 0 THEN 'deducted'
        WHEN count(*) FILTER (WHERE current_step = 'verifying_stock') > 0 THEN 'pending'
        WHEN count(*) FILTER (WHERE current_step = 'failed') > 0 THEN 'rejected'
        ELSE ''
    END AS status
    FROM saga_state WHERE order_id = o.id AND o.status = 'paid'
) AS stock
WHERE o.id = $1;

-- name: ListAssignedOrders :many
SELECT o.id, o.vehicle_id, o.status, o.stage, o.created_at,
       (o.paid_at IS NOT NULL OR EXISTS (
           SELECT 1 FROM saga_state AS s
           WHERE s.order_id = o.id AND s.current_step = 'completed'
       )) AS payment_confirmed
FROM orders AS o
WHERE o.engineer_id = sqlc.arg(engineer_id)
  AND CASE WHEN sqlc.arg(completed)::boolean
      THEN o.status = 'cancelled' OR (o.status = 'paid' AND o.stage = 'выдан')
      ELSE o.status = 'active' OR (o.status = 'paid' AND o.stage <> 'выдан')
  END
ORDER BY o.created_at DESC, o.id DESC;

-- name: GetVehicleForEngineerHistory :one
-- Заказ даёт доступ к истории машины только текущему ответственному до выдачи.
SELECT vehicle_id
FROM orders
WHERE id = sqlc.arg(order_id)
  AND engineer_id = sqlc.arg(engineer_id)
  AND (status = 'active' OR (status = 'paid' AND stage <> 'выдан'));

-- name: ListVehicleOrderHistory :many
-- Вызывается в том же снимке БД, что и GetVehicleForEngineerHistory.
SELECT o.id, o.vehicle_id, o.status, o.stage, o.created_at,
       (o.paid_at IS NOT NULL OR EXISTS (
           SELECT 1 FROM saga_state AS s
           WHERE s.order_id = o.id AND s.current_step = 'completed'
       )) AS payment_confirmed
FROM orders AS o
WHERE o.vehicle_id = sqlc.arg(vehicle_id)
  AND o.id <> sqlc.arg(current_order_id)
ORDER BY o.created_at DESC, o.id DESC;

-- name: ListOrderItems :many
SELECT id, item_id, name, quantity, price
FROM order_items
WHERE order_id = $1
ORDER BY created_at, id;

-- name: ChangeOrderStage :execrows
UPDATE orders
SET stage = sqlc.arg(stage), updated_at = now()
WHERE id = sqlc.arg(id)
  AND engineer_id = sqlc.arg(engineer_id)::uuid
  AND (sqlc.arg(stage)::text <> 'выдан' OR (
      status = 'paid' AND EXISTS (
          SELECT 1 FROM saga_state WHERE order_id = orders.id AND current_step = 'completed'
      )
  ));

-- name: CancelOrder :execrows
UPDATE orders SET status = 'cancelled', updated_at = now()
WHERE id = sqlc.arg(id) AND engineer_id = sqlc.arg(engineer_id)::uuid AND status = 'active';

-- name: AddOrderService :one
INSERT INTO order_services (order_id, service_id, name, quantity, price)
SELECT o.id, sqlc.arg(service_id), sqlc.arg(name), sqlc.arg(quantity), sqlc.arg(price)
FROM orders AS o
WHERE o.id = sqlc.arg(order_id) AND o.engineer_id = sqlc.arg(engineer_id)::uuid AND o.status = 'active'
FOR UPDATE OF o
RETURNING id;

-- name: ListOrderServices :many
SELECT id, service_id, name, quantity, price
FROM order_services WHERE order_id = $1 ORDER BY created_at, id;
