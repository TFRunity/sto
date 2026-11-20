-- name: CreateVehicle :one
INSERT INTO vehicles (owner_id, number, brand, vin)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: GetVehicle :one
SELECT id, owner_id, number, brand, vin
FROM vehicles
WHERE id = $1 AND deleted_at IS NULL;
