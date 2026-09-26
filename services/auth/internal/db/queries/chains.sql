-- name: EnsureChain :one
INSERT INTO chains (id)
VALUES ($1)
ON CONFLICT (id) DO UPDATE
SET id = EXCLUDED.id
RETURNING revoked_at;

-- name: TombstoneChain :exec
INSERT INTO chains (id, revoked_at)
VALUES ($1, NOW())
ON CONFLICT (id) DO UPDATE
SET revoked_at = COALESCE(chains.revoked_at, EXCLUDED.revoked_at);

-- name: DeleteOrphanChains :execrows
DELETE FROM chains c
WHERE c.created_at < NOW() - interval '1 day'
  AND NOT EXISTS (SELECT 1 FROM refresh_tokens r WHERE r.chain_id = c.id);

-- name: LockChain :one
UPDATE chains
SET id = id
WHERE id = $1
RETURNING revoked_at;
