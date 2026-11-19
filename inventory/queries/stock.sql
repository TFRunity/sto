-- name: ClaimStockAttempt :execrows
INSERT INTO stock_attempts (saga_id, order_id, state)
VALUES ($1, $2, $3) ON CONFLICT (saga_id) DO NOTHING;

-- name: LockStockAttempt :one
SELECT state, order_id FROM stock_attempts WHERE saga_id = $1 FOR UPDATE;

-- name: MarkStockAttempt :execrows
UPDATE stock_attempts SET state = $2, updated_at = now()
WHERE saga_id = $1 AND state = $3;

-- name: DeductItem :execrows
UPDATE inventory SET quantity = quantity - sqlc.arg(qty)::bigint, updated_at = now()
WHERE id = sqlc.arg(id) AND quantity >= sqlc.arg(qty)::bigint AND deleted_at IS NULL;

-- name: RecordDeductedItem :exec
INSERT INTO stock_attempt_items (saga_id, item_id, quantity)
VALUES ($1, $2, $3);

-- include-deleted: вернуть ранее списанное, даже если товар сняли с продажи.
-- name: RestoreStock :exec
UPDATE inventory AS i SET quantity = i.quantity + a.quantity, updated_at = now()
FROM stock_attempt_items AS a WHERE a.saga_id = $1 AND i.id = a.item_id;
