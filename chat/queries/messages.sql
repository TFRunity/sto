-- name: CreateMessage :one
-- Отправитель берётся из доверенного контекста. Для диалога по заказу
-- вызывающий код сначала проверяет актуальное назначение в Orders.
INSERT INTO messages (dialog_id, sender_id, body)
SELECT d.id, sqlc.arg(sender_id)::uuid, sqlc.arg(body)::text
FROM dialogs AS d
WHERE d.id = sqlc.arg(dialog_id)::uuid
    AND sqlc.arg(sender_id)::uuid IN (d.participant_1_id, d.participant_2_id)
RETURNING id, dialog_id, sender_id, body, is_read, created_at;

-- name: ListDialogMessages :many
SELECT id, dialog_id, sender_id, body, is_read, created_at
FROM messages
WHERE dialog_id = $1
ORDER BY created_at, id;

-- name: MarkDialogReadThrough :one
-- Сообщение ограничивает отметку тем, что собеседник уже получил в истории.
WITH boundary AS (
    SELECT created_at, id
    FROM messages
    WHERE dialog_id = sqlc.arg(dialog_id)::uuid AND id = sqlc.arg(through_message_id)::uuid
), marked AS (
    UPDATE messages AS m
    SET is_read = true
    FROM boundary AS b
    WHERE m.dialog_id = sqlc.arg(dialog_id)::uuid
        AND m.sender_id <> sqlc.arg(reader_id)::uuid
        AND NOT m.is_read
        AND (m.created_at, m.id) <= (b.created_at, b.id)
    RETURNING m.id
)
SELECT EXISTS (SELECT 1 FROM boundary) AS boundary_exists,
       EXISTS (SELECT 1 FROM marked) AS read_changed;
