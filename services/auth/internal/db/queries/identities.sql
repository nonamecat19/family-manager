-- name: GetIdentity :one
SELECT * FROM identities
WHERE provider = $1 AND external_id = $2;

-- name: UpsertIdentity :one
INSERT INTO identities (user_id, provider, external_id, chain_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (provider, external_id) DO UPDATE
SET chain_id   = EXCLUDED.chain_id,
    updated_at = NOW()
WHERE identities.user_id = EXCLUDED.user_id
RETURNING *;

-- name: ListIdentitiesForUser :many
SELECT * FROM identities
WHERE user_id = $1
ORDER BY provider, created_at;

-- name: DeleteIdentity :one
DELETE FROM identities
WHERE user_id = $1 AND provider = $2 AND external_id = $3
RETURNING *;

-- name: LockIdentityKey :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(key)::text, 0));
