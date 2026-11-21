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
WHERE c.order_id = $1 AND o.client_id = $2 AND c.status = 'completed';

-- name: ListCertificateEngineerIDs :many
SELECT engineer_id FROM order_engineers
WHERE order_id = $1
ORDER BY created_at, engineer_id;
