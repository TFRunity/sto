-- +goose Up
ALTER TABLE orders ADD COLUMN paid_at timestamptz;

-- Для v1 факт оплаты подтверждался успешным завершением саги.
UPDATE orders AS o
SET paid_at = s.confirmed_at
FROM (
    SELECT order_id, min(updated_at) AS confirmed_at
    FROM saga_state
    WHERE current_step = 'completed'
    GROUP BY order_id
) AS s
WHERE o.id = s.order_id AND o.status = 'paid';
