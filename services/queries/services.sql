-- name: GetService :one
SELECT id, name, price FROM services WHERE id = $1 AND deleted_at IS NULL;

-- name: ListServices :many
SELECT id, name, price FROM services
WHERE deleted_at IS NULL ORDER BY name, id;

-- name: CreateService :one
INSERT INTO services (name, price) VALUES ($1, $2) RETURNING id, name, price;

-- name: UpdateService :one
UPDATE services SET name = $2, price = $3, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL RETURNING id, name, price;

-- name: DeleteService :execrows
UPDATE services SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
