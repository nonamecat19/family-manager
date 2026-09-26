-- name: UpsertLink :one
INSERT INTO telegram_links (
    telegram_user_id, user_id, telegram_username, chat_id,
    access_token, access_expires_at, refresh_token
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (telegram_user_id) DO UPDATE
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
WHERE telegram_user_id = $1;

-- name: UpdateLinkTokens :execrows
UPDATE telegram_links
SET access_token      = sqlc.arg(access_token),
    access_expires_at = sqlc.arg(access_expires_at),
    refresh_token     = sqlc.arg(refresh_token),
    updated_at        = NOW()
WHERE telegram_user_id = sqlc.arg(telegram_user_id)
  AND refresh_token = sqlc.arg(previous_refresh_token);

-- name: DeleteLinkWithToken :execrows
DELETE FROM telegram_links
WHERE telegram_user_id = $1 AND refresh_token = $2;

-- name: DeleteLinkForUser :execrows
DELETE FROM telegram_links
WHERE telegram_user_id = $1 AND user_id = $2;

-- name: ListLinksForUser :many
SELECT * FROM telegram_links
WHERE user_id = $1
ORDER BY telegram_user_id;

-- name: ExpireLinkAccess :execrows
UPDATE telegram_links
SET access_expires_at = NOW() - interval '1 second',
    updated_at        = NOW()
WHERE telegram_user_id = $1;
