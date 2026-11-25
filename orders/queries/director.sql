-- name: ListOrdersForDirector :many
WITH page AS (
    SELECT id, client_id, vehicle_id, engineer_id, status, stage, created_at
    FROM orders
    WHERE (
        sqlc.narg(status)::text IS NULL
        OR status = sqlc.narg(status)::text
        OR (sqlc.narg(status)::text = 'awaiting_payment' AND status = 'active' AND stage = 'готов')
    )
    AND (
        (sqlc.narg(client_ids)::uuid[] IS NULL AND sqlc.narg(vehicle_ids)::uuid[] IS NULL)
        OR client_id = ANY(sqlc.narg(client_ids)::uuid[])
        OR vehicle_id = ANY(sqlc.narg(vehicle_ids)::uuid[])
    )
    ORDER BY created_at DESC, id DESC
    LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::bigint
)
SELECT p.*,
       (COALESCE((SELECT sum(quantity * price) FROM order_items WHERE order_id = p.id), 0)
        + COALESCE((SELECT sum(quantity * price) FROM order_services WHERE order_id = p.id), 0))::numeric AS total
FROM page AS p
ORDER BY p.created_at DESC, p.id DESC;

-- name: GetOrderStats :one
WITH bounds AS (
    SELECT date_trunc('day', now() AT TIME ZONE 'Europe/Moscow') AS day
)
SELECT count(*) FILTER (WHERE o.created_at >= (b.day AT TIME ZONE 'Europe/Moscow')
                        AND o.created_at < ((b.day + interval '1 day') AT TIME ZONE 'Europe/Moscow')) AS today_orders,
       count(*) FILTER (WHERE o.status = 'active') AS active_orders,
       count(*) FILTER (WHERE o.status = 'active' AND o.stage = 'готов') AS awaiting_payment,
       COALESCE(sum(
           COALESCE((SELECT sum(quantity * price) FROM order_items WHERE order_id = o.id), 0)
           + COALESCE((SELECT sum(quantity * price) FROM order_services WHERE order_id = o.id), 0)
       ) FILTER (WHERE o.status = 'paid'
           AND COALESCE(o.paid_at, legacy.confirmed_at) >= (b.day AT TIME ZONE 'Europe/Moscow')
           AND COALESCE(o.paid_at, legacy.confirmed_at) < ((b.day + interval '1 day') AT TIME ZONE 'Europe/Moscow')
       ), 0)::numeric AS today_revenue
FROM orders AS o CROSS JOIN bounds AS b
LEFT JOIN (
    SELECT order_id, min(updated_at) AS confirmed_at
    FROM saga_state WHERE current_step = 'completed'
    GROUP BY order_id
) AS legacy ON o.id = legacy.order_id AND o.paid_at IS NULL;

-- name: CountActiveOrdersByEngineer :many
SELECT engineer_id, count(*) AS active_orders FROM orders
WHERE engineer_id IS NOT NULL AND status = 'active'
GROUP BY engineer_id;
