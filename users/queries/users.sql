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
