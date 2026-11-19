-- name: PayOrder :execrows
UPDATE orders SET status = 'paid', updated_at = now()
WHERE id = $1 AND client_id = $2 AND status = 'active';

-- name: CreateSaga :exec
INSERT INTO saga_state (saga_id, order_id, current_step, deadline)
VALUES ($1, $2, 'verifying_stock', clock_timestamp() + interval '10 seconds');

-- name: ListDeductionItems :many
SELECT item_id, sum(quantity)::bigint AS quantity
FROM order_items WHERE order_id = $1 GROUP BY item_id ORDER BY item_id;

-- name: GetSaga :one
SELECT order_id, current_step FROM saga_state WHERE saga_id = $1;

-- name: TransitionSaga :execrows
UPDATE saga_state SET current_step = $2, updated_at = now()
WHERE saga_id = $1 AND current_step = 'verifying_stock' AND deadline > clock_timestamp();

-- name: ExpireSaga :execrows
UPDATE saga_state SET current_step = 'failed', updated_at = now()
WHERE saga_id = $1 AND current_step = 'verifying_stock' AND deadline <= clock_timestamp();

-- name: RestoreActiveOrder :exec
UPDATE orders SET status = 'active', updated_at = now()
WHERE id = $1 AND status = 'paid';

-- name: CreateCertificate :exec
INSERT INTO certificates (order_id, status, next_attempt_at)
VALUES ($1, 'pending', now());

-- name: ListExpiredSagas :many
SELECT saga_id, order_id FROM saga_state
WHERE current_step = 'verifying_stock' AND deadline <= now()
ORDER BY deadline, saga_id LIMIT 50;
