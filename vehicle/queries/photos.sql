-- name: ReplaceVehicleCoverPhoto :execrows
UPDATE vehicles
SET cover_photo_url = sqlc.arg(file_key), updated_at = now()
WHERE id = sqlc.arg(vehicle_id) AND owner_id = sqlc.arg(owner_id) AND deleted_at IS NULL;

-- FOR SHARE удерживает строку ТС до завершения вставки снимка,
-- чтобы параллельное мягкое удаление дождалось этой операции.

-- name: AddVehiclePhoto :one
INSERT INTO vehicle_photos (vehicle_id, order_id, file_key)
SELECT v.id, sqlc.narg(order_id)::uuid, sqlc.arg(file_key)::text
FROM vehicles AS v
WHERE v.id = sqlc.arg(vehicle_id) AND v.deleted_at IS NULL
FOR SHARE OF v
RETURNING id;

-- name: ListVehiclePhotoIDs :many
SELECT p.id
FROM vehicle_photos AS p
JOIN vehicles AS v ON v.id = p.vehicle_id
WHERE p.vehicle_id = $1 AND v.deleted_at IS NULL
ORDER BY p.created_at, p.id;

-- name: GetVehiclePhotoKey :one
SELECT p.file_key
FROM vehicle_photos AS p
JOIN vehicles AS v ON v.id = p.vehicle_id
WHERE p.id = sqlc.arg(photo_id) AND p.vehicle_id = sqlc.arg(vehicle_id) AND v.deleted_at IS NULL;
