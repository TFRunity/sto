-- name: ListDueCertificates :many
SELECT id, order_id, status, attempts FROM certificates
WHERE status IN ('pending', 'failed') AND attempts < 3 AND next_attempt_at <= now()
ORDER BY next_attempt_at, id LIMIT 20;

-- name: SaveCertificateAttempt :execrows
UPDATE certificates
SET status = sqlc.arg(new_status), attempts = sqlc.arg(new_attempts),
    next_attempt_at = sqlc.narg(next_attempt_at), snapshot = sqlc.narg(snapshot),
    pdf = sqlc.narg(pdf), updated_at = now()
WHERE id = sqlc.arg(id) AND status = sqlc.arg(old_status) AND attempts = sqlc.arg(old_attempts);

-- name: GetCompletedCertificate :one
SELECT c.pdf FROM certificates AS c JOIN orders AS o ON o.id = c.order_id
WHERE c.order_id = sqlc.arg(order_id) AND c.status = 'completed'
  AND (sqlc.arg(is_director)::boolean OR o.client_id = sqlc.arg(client_id)::uuid);

-- name: ListClientCertificates :many
SELECT c.id, c.order_id, c.created_at, c.status, c.attempts, c.next_attempt_at
FROM certificates AS c
JOIN orders AS o ON o.id = c.order_id
WHERE o.client_id = sqlc.arg(client_id)
ORDER BY c.created_at DESC, c.id DESC;

-- name: GetOrderCertificateState :one
SELECT status, attempts, next_attempt_at FROM certificates WHERE order_id = $1;

-- name: ListCertificateEngineerIDs :many
SELECT engineer_id FROM order_engineers
WHERE order_id = $1
ORDER BY created_at, engineer_id;
