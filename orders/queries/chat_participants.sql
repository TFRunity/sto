-- name: GetOrderChatParticipants :one
SELECT client_id, engineer_id
FROM orders
WHERE id = $1;
