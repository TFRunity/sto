-- name: CreateClient :one
-- Регистрация через API создаёт только client (ADR-019); engineer / director — seed.
-- Занятый телефон → unique_violation по uq_users_phone → 409 phone_taken.
INSERT INTO users (id, name, phone, password_hash, role)
VALUES (@id, @name, @phone, @password_hash, 'client')
RETURNING *;

-- name: GetUserByPhone :one
-- Вход: хеш и роль для JWT. Только живые пользователи (ADR-033).
SELECT *
FROM users
WHERE phone = @phone
  AND deleted_at IS NULL;

-- include-deleted: директор ищет клиентов из истории заказов.
-- name: SearchClients :many
SELECT id FROM users
WHERE role = 'client' AND (
    name ILIKE '%' || sqlc.arg(pattern)::text || '%' ESCAPE '!'
    OR phone ILIKE '%' || sqlc.arg(pattern)::text || '%' ESCAPE '!'
)
ORDER BY id LIMIT 201;

-- include-deleted: подписи к историческим заказам и справкам.
-- name: GetPeopleByIDs :many
SELECT id, name, phone FROM users WHERE id = ANY(sqlc.arg(ids)::uuid[]);

-- name: ListEngineersForLoad :many
SELECT id, name, phone FROM users
WHERE role = 'engineer' AND deleted_at IS NULL
ORDER BY name, id;

-- name: SearchActiveClients :many
SELECT id, name, phone
FROM users
WHERE role = 'client' AND deleted_at IS NULL AND (
    name ILIKE '%' || sqlc.arg(pattern)::text || '%' ESCAPE '!'
    OR phone ILIKE '%' || sqlc.arg(pattern)::text || '%' ESCAPE '!'
)
ORDER BY name, id LIMIT 201;

-- name: GetMyProfile :one
SELECT name, phone, role, created_at, work_started_at, notifications_enabled
FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateMyNotifications :one
UPDATE users
SET notifications_enabled = sqlc.arg(enabled), updated_at = now()
WHERE id = sqlc.arg(user_id) AND deleted_at IS NULL
RETURNING name, phone, role, created_at, work_started_at, notifications_enabled;
