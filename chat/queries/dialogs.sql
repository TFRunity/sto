-- name: OpenOrderDialog :one
-- Обновление ID самим собой возвращает существующий ID и при конкурентном открытии.
INSERT INTO dialogs (kind, participant_1_id, participant_2_id, order_id)
VALUES ('order', sqlc.arg(client_id)::uuid, sqlc.arg(engineer_id)::uuid, sqlc.arg(order_id)::uuid)
ON CONFLICT (kind, (LEAST(participant_1_id, participant_2_id)),
             (GREATEST(participant_1_id, participant_2_id)), order_id)
DO UPDATE SET id = dialogs.id
RETURNING id;

-- name: GetParticipantDialog :one
SELECT id, kind, participant_1_id, participant_2_id, order_id, created_at
FROM dialogs
WHERE id = sqlc.arg(dialog_id)::uuid
    AND sqlc.arg(user_id)::uuid IN (participant_1_id, participant_2_id);

-- name: ListParticipantDialogs :many
SELECT d.id, d.kind, d.participant_1_id, d.participant_2_id, d.order_id, d.created_at,
    COALESCE(stats.last_message_at, d.created_at)::timestamptz AS last_activity_at,
    stats.unread_count
FROM dialogs AS d
CROSS JOIN LATERAL (
    SELECT max(m.created_at) AS last_message_at,
        count(*) FILTER (WHERE m.sender_id <> sqlc.arg(user_id)::uuid AND NOT m.is_read) AS unread_count
    FROM messages AS m
    WHERE m.dialog_id = d.id
) AS stats
WHERE sqlc.arg(user_id)::uuid IN (d.participant_1_id, d.participant_2_id)
ORDER BY last_activity_at DESC, d.id DESC;

-- name: OpenDirectorEngineerDialog :one
INSERT INTO dialogs (kind, participant_1_id, participant_2_id)
VALUES ('director_engineer', sqlc.arg(director_id)::uuid, sqlc.arg(engineer_id)::uuid)
ON CONFLICT (kind, (LEAST(participant_1_id, participant_2_id)),
             (GREATEST(participant_1_id, participant_2_id)), order_id)
DO UPDATE SET id = dialogs.id
RETURNING id;

-- name: OpenSupportDialog :one
INSERT INTO dialogs (kind, participant_1_id, participant_2_id)
VALUES ('support', sqlc.arg(client_id)::uuid, sqlc.arg(director_id)::uuid)
ON CONFLICT (kind, (LEAST(participant_1_id, participant_2_id)),
             (GREATEST(participant_1_id, participant_2_id)), order_id)
DO UPDATE SET id = dialogs.id
RETURNING id;
