-- name: UpsertLink :one
INSERT INTO telegram_links (
    bot, telegram_user_id, user_id, telegram_username, chat_id,
    access_token, access_expires_at, refresh_token
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (bot, telegram_user_id) DO UPDATE
SET user_id           = EXCLUDED.user_id,
    telegram_username = EXCLUDED.telegram_username,
    chat_id           = EXCLUDED.chat_id,
    access_token      = EXCLUDED.access_token,
    access_expires_at = EXCLUDED.access_expires_at,
    refresh_token     = EXCLUDED.refresh_token,
    updated_at        = NOW()
RETURNING *;

-- name: GetLink :one
SELECT * FROM telegram_links
WHERE bot = $1 AND telegram_user_id = $2;

-- name: UpdateLinkTokens :one
UPDATE telegram_links
SET access_token      = $3,
    access_expires_at = $4,
    refresh_token     = $5,
    updated_at        = NOW()
WHERE bot = $1 AND telegram_user_id = $2
RETURNING *;

-- name: DeleteLink :execrows
DELETE FROM telegram_links
WHERE bot = $1 AND telegram_user_id = $2;

-- name: ListLinksForUser :many
SELECT * FROM telegram_links
WHERE user_id = $1
ORDER BY bot;

-- name: ExpireLinkAccess :execrows
UPDATE telegram_links
SET access_expires_at = NOW() - interval '1 second',
    updated_at        = NOW()
WHERE bot = $1 AND telegram_user_id = $2;
