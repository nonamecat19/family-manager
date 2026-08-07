-- The WHERE NOT EXISTS closes a race: a refresh that passed the "is this chain alive?" check
-- can otherwise insert its successor a moment after a concurrent replay revoked the chain,
-- resurrecting it. Inserting nothing returns no rows, which the handler treats as a refusal.
--
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

-- Marks the token spent. The WHERE clause is the concurrency guard: two refreshes racing on
-- the same token, only one updates a row, and the loser is treated as a replay.
--
-- name: MarkRefreshTokenUsed :execrows
UPDATE refresh_tokens
SET used_at = NOW()
WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL;

-- Reuse of a spent token means it leaked: kill the whole rotation chain, not just this token,
-- because the thief and the victim both hold descendants of it.
--
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
