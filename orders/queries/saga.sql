-- name: ConfirmOrderPayment :one
UPDATE orders
SET status = 'paid', paid_at = now(), updated_at = now()
WHERE id = $1 AND client_id = $2 AND status = 'active'
RETURNING paid_at;

-- name: GetConfirmedPaymentTime :one
SELECT paid_at FROM orders
WHERE id = $1 AND client_id = $2 AND status = 'paid' AND paid_at IS NOT NULL;

-- name: CreateStockAttempt :exec
INSERT INTO saga_state (saga_id, order_id, current_step, deadline)
VALUES ($1, $2, 'verifying_stock', 'infinity'::timestamptz);

-- name: LockPaidOrderForStockRetry :one
SELECT id FROM orders
WHERE id = $1 AND status = 'paid' AND paid_at IS NOT NULL
FOR UPDATE;

-- name: CanRetryStockDeduction :one
SELECT EXISTS (
    SELECT 1 FROM saga_state AS failed_attempt
    WHERE failed_attempt.order_id = sqlc.arg(order_id) AND failed_attempt.current_step = 'failed'
) AND NOT EXISTS (
    SELECT 1 FROM saga_state AS blocking_attempt
    WHERE blocking_attempt.order_id = sqlc.arg(order_id)
      AND blocking_attempt.current_step IN ('verifying_stock', 'completed')
) AS can_retry;

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
WHERE id = $1 AND status = 'paid' AND paid_at IS NULL;

-- name: CreateCertificate :exec
INSERT INTO certificates (order_id, status, next_attempt_at)
VALUES ($1, 'pending', now())
ON CONFLICT (order_id) DO NOTHING;

-- name: ListExpiredSagas :many
SELECT saga_id, order_id FROM saga_state
WHERE current_step = 'verifying_stock' AND deadline <= now()
ORDER BY deadline, saga_id LIMIT 50;
