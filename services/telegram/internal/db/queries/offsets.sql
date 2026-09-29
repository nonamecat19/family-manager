-- name: GetBotOffset :one
SELECT offset_id FROM bot_offsets
WHERE bot = $1;

-- name: SetBotOffset :exec
INSERT INTO bot_offsets (bot, offset_id)
VALUES ($1, $2)
ON CONFLICT (bot) DO UPDATE
SET offset_id = EXCLUDED.offset_id, updated_at = NOW();
