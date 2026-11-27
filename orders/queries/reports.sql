-- name: ListRevenueReportOrders :many
SELECT o.id AS order_id, o.paid_at,
       (COALESCE((SELECT sum(quantity * price) FROM order_items WHERE order_id = o.id), 0)
        + COALESCE((SELECT sum(quantity * price) FROM order_services WHERE order_id = o.id), 0))::numeric AS total_amount
FROM orders AS o
WHERE o.status = 'paid'
    AND o.paid_at >= sqlc.arg(period_from)::timestamptz
    AND o.paid_at < sqlc.arg(period_to)::timestamptz
ORDER BY o.paid_at, o.id;

-- name: ListEngineersReportOrders :many
-- Периоды создания и оплаты могут пересекаться; один заказ даёт одну строку.
SELECT o.id AS order_id, o.created_at, o.paid_at, o.status,
       ARRAY(SELECT oe.engineer_id FROM order_engineers AS oe
             WHERE oe.order_id = o.id ORDER BY oe.engineer_id)::uuid[] AS engineer_ids,
       (COALESCE((SELECT sum(quantity * price) FROM order_items WHERE order_id = o.id), 0)
        + COALESCE((SELECT sum(quantity * price) FROM order_services WHERE order_id = o.id), 0))::numeric AS total_amount
FROM orders AS o
WHERE (o.created_at >= sqlc.arg(period_from)::timestamptz AND o.created_at < sqlc.arg(period_to)::timestamptz)
   OR (o.paid_at >= sqlc.arg(period_from)::timestamptz AND o.paid_at < sqlc.arg(period_to)::timestamptz)
ORDER BY o.created_at, o.id;

-- name: ListCertificatesReportOrders :many
SELECT o.id AS order_id, o.created_at, o.status, o.stage,
       c.id AS certificate_id, c.status AS certificate_status,
       c.created_at AS certificate_created_at
FROM orders AS o
LEFT JOIN certificates AS c ON c.order_id = o.id
WHERE (o.created_at >= sqlc.arg(period_from)::timestamptz AND o.created_at < sqlc.arg(period_to)::timestamptz)
   OR (c.created_at >= sqlc.arg(period_from)::timestamptz AND c.created_at < sqlc.arg(period_to)::timestamptz)
ORDER BY o.created_at, o.id;
