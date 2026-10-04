-- name: CreateLoginGrant :one
INSERT INTO login_grants (kind, device_code_hash, user_code_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetLoginGrantByDeviceCode :one
SELECT * FROM login_grants
WHERE device_code_hash = $1;

-- name: GetLoginGrantByUserCode :one
SELECT * FROM login_grants
WHERE user_code_hash = $1;

-- name: TouchLoginGrant :execrows
UPDATE login_grants
SET last_polled_at = sqlc.arg(polled_at)
WHERE id = sqlc.arg(id)
  AND (last_polled_at IS NULL OR last_polled_at <= sqlc.arg(not_after));

-- name: ApproveLoginGrant :one
UPDATE login_grants g
SET user_id           = sqlc.arg(user_id),
    approved_at       = sqlc.arg(decided_at),
    approver_chain_id = sqlc.narg(approver_chain_id),
    root_chain_id     = COALESCE(
      (SELECT p.root_chain_id FROM login_grants p
       WHERE p.chain_id = sqlc.narg(approver_chain_id)::uuid),
      sqlc.narg(approver_chain_id)::uuid
    )
WHERE g.user_code_hash = sqlc.arg(user_code_hash)
  AND g.approved_at IS NULL
  AND g.denied_at IS NULL
  AND g.expires_at > sqlc.arg(decided_at)
RETURNING *;

-- name: DenyLoginGrant :execrows
UPDATE login_grants
SET user_id = sqlc.arg(user_id), denied_at = sqlc.arg(decided_at)
WHERE user_code_hash = sqlc.arg(user_code_hash)
  AND approved_at IS NULL
  AND denied_at IS NULL
  AND expires_at > sqlc.arg(decided_at);

-- name: ConsumeLoginGrant :execrows
UPDATE login_grants g
SET consumed_at = NOW(), chain_id = sqlc.arg(chain_id)
WHERE g.id = sqlc.arg(id)
  AND g.approved_at IS NOT NULL
  AND g.denied_at IS NULL
  AND g.consumed_at IS NULL
  AND g.expires_at > NOW()
  AND (
    g.approver_chain_id IS NULL
    OR (
      EXISTS (
        SELECT 1 FROM chains c
        WHERE c.id = g.root_chain_id AND c.revoked_at IS NULL
        FOR SHARE
      )
      AND EXISTS (
        SELECT 1 FROM chains c
        WHERE c.id = g.approver_chain_id AND c.revoked_at IS NULL
        FOR SHARE
      )
    )
  );

-- name: CountPendingLoginGrants :one
SELECT COUNT(*) FROM login_grants
WHERE consumed_at IS NULL
  AND denied_at IS NULL
  AND expires_at > NOW();

-- name: TombstoneChainsRootedAt :exec
INSERT INTO chains (id, revoked_at)
SELECT DISTINCT g.chain_id, NOW()
FROM login_grants g
WHERE g.root_chain_id = $1 AND g.chain_id IS NOT NULL
ON CONFLICT (id) DO UPDATE
SET revoked_at = COALESCE(chains.revoked_at, EXCLUDED.revoked_at);

-- name: RevokeChainsRootedAt :execrows
UPDATE refresh_tokens r
SET revoked_at = NOW()
WHERE r.revoked_at IS NULL
  AND r.chain_id IN (
    SELECT g.chain_id FROM login_grants g
    WHERE g.root_chain_id = $1 AND g.chain_id IS NOT NULL
  );

-- name: DeleteExpiredLoginGrants :execrows
DELETE FROM login_grants g
WHERE g.expires_at < NOW() - interval '1 hour'
  AND NOT EXISTS (
    SELECT 1 FROM chains c
    WHERE c.id = g.chain_id AND c.revoked_at IS NULL
  );

-- name: GetChainRoot :one
SELECT COALESCE(
  (SELECT g.root_chain_id FROM login_grants g WHERE g.chain_id = sqlc.arg(chain_id)::uuid),
  sqlc.arg(chain_id)::uuid
)::uuid AS root_chain_id;
