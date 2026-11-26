-- name: RegisterPushDevice :execrows
-- Проверяем актуальную роль и soft delete даже при действующем JWT.
INSERT INTO push_devices (token, user_id)
SELECT sqlc.arg(token), id
FROM users
WHERE id = sqlc.arg(user_id) AND role = 'engineer' AND deleted_at IS NULL
ON CONFLICT (token) DO UPDATE
SET user_id = EXCLUDED.user_id, registered_at = clock_timestamp();

-- name: RevokePushDevice :exec
DELETE FROM push_devices
WHERE token = sqlc.arg(token) AND user_id = sqlc.arg(user_id);

-- name: ListBookingPushDevices :many
SELECT d.token, d.user_id, d.registered_at
FROM push_devices AS d
JOIN users AS u ON u.id = d.user_id
WHERE u.role = 'engineer'
    AND u.deleted_at IS NULL
    AND u.notifications_enabled;

-- name: ListChatPushDevices :many
SELECT d.token, d.user_id, d.registered_at
FROM push_devices AS d
JOIN users AS u ON u.id = d.user_id
WHERE u.id = sqlc.arg(recipient_id)
    AND u.role = 'engineer'
    AND u.deleted_at IS NULL
    AND u.notifications_enabled;

-- name: DeleteInvalidPushDevice :exec
-- Запоздалый ответ провайдера не удаляет обновлённую регистрацию.
DELETE FROM push_devices
WHERE token = sqlc.arg(token) AND registered_at = sqlc.arg(registered_at);
