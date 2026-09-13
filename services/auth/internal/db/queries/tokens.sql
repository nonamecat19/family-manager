-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, chain_id, expires_at)
SELECT $1, $2, $3, $4
WHERE NOT EXISTS (
    SELECT 1 FROM refresh_tokens
    WHERE chain_id = $3 AND revoked_at IS NOT NULL
)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1;

-- name: MarkRefreshTokenUsed :execrows
UPDATE refresh_tokens
SET used_at = NOW()
WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL;

-- name: RevokeChain :execrows
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE chain_id = $1 AND revoked_at IS NULL;

-- name: RevokeAllForUser :execrows
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredRefreshTokens :execrows
DELETE FROM refresh_tokens
WHERE expires_at < NOW();
