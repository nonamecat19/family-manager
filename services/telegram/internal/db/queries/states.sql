-- name: UpsertChatState :one
INSERT INTO chat_states (bot, telegram_user_id, kind, payload, expires_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (bot, telegram_user_id) DO UPDATE
SET kind       = EXCLUDED.kind,
    payload    = EXCLUDED.payload,
    expires_at = EXCLUDED.expires_at,
    updated_at = NOW()
RETURNING *;

-- name: GetChatState :one
SELECT * FROM chat_states
WHERE bot = $1 AND telegram_user_id = $2 AND expires_at > NOW();

-- name: ClearChatState :execrows
DELETE FROM chat_states
WHERE bot = $1 AND telegram_user_id = $2;

-- name: DeleteExpiredChatStates :execrows
DELETE FROM chat_states
WHERE expires_at < NOW();
