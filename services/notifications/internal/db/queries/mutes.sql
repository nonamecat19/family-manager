-- name: ListMutes :many
SELECT topic FROM notification_mutes
WHERE user_id = $1
ORDER BY topic;

-- name: ListMutesForUsers :many
SELECT user_id, topic FROM notification_mutes
WHERE user_id = ANY(@user_ids::uuid[]);

-- name: DeleteMutes :exec
DELETE FROM notification_mutes
WHERE user_id = $1;

-- name: InsertMute :exec
INSERT INTO notification_mutes (user_id, topic)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
