-- name: CreateVehicle :one
INSERT INTO vehicles (owner_id, number, brand, vin)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: GetVehicle :one
SELECT id, owner_id, number, brand, vin, cover_photo_url
FROM vehicles
WHERE id = $1 AND deleted_at IS NULL;

-- include-deleted: директор ищет ТС из истории заказов.
-- name: SearchVehicles :many
SELECT id FROM vehicles
WHERE number ILIKE '%' || sqlc.arg(pattern)::text || '%' ESCAPE '!'
ORDER BY id LIMIT 201;

-- include-deleted: номера ТС из истории заказов.
-- name: GetVehiclesByIDs :many
SELECT id, number FROM vehicles WHERE id = ANY(sqlc.arg(ids)::uuid[]);
