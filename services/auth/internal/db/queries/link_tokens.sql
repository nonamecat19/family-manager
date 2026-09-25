-- name: CreateLinkToken :one
INSERT INTO link_tokens (user_id, provider, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetLinkToken :one
SELECT * FROM link_tokens
WHERE token_hash = $1;

-- name: MarkLinkTokenUsed :execrows
UPDATE link_tokens
SET used_at = NOW(), external_id = $2
WHERE id = $1 AND used_at IS NULL;

-- name: DeleteExpiredLinkTokens :execrows
DELETE FROM link_tokens
WHERE expires_at < NOW();
