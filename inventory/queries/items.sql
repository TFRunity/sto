-- name: GetItem :one
SELECT id, name, price
FROM inventory
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListItems :many
SELECT id, name, price, quantity FROM inventory
WHERE deleted_at IS NULL ORDER BY name, id;

-- name: CreateItem :one
INSERT INTO inventory (name, price, quantity)
VALUES ($1, $2, $3) RETURNING id, name, price, quantity;

-- name: UpdateItem :one
UPDATE inventory SET name = $2, price = $3, quantity = $4, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, name, price, quantity;

-- name: DeleteItem :execrows
UPDATE inventory SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
