-- name: GetItem :one
SELECT id, name, price
FROM inventory
WHERE id = $1 AND deleted_at IS NULL;
