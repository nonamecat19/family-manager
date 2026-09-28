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
UPDATE login_grants
SET user_id           = sqlc.arg(user_id),
    approved_at       = sqlc.arg(decided_at),
    approver_chain_id = sqlc.narg(approver_chain_id)
WHERE user_code_hash = sqlc.arg(user_code_hash)
  AND approved_at IS NULL
  AND denied_at IS NULL
  AND expires_at > sqlc.arg(decided_at)
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
    OR EXISTS (
      SELECT 1 FROM chains c
      WHERE c.id = g.approver_chain_id AND c.revoked_at IS NULL
      FOR SHARE
    )
  );

-- name: CountPendingLoginGrants :one
SELECT COUNT(*) FROM login_grants
WHERE consumed_at IS NULL
  AND denied_at IS NULL
  AND expires_at > NOW();

-- name: ListChainsApprovedFrom :many
WITH RECURSIVE lineage (id) AS (
  SELECT g.chain_id FROM login_grants g
  WHERE g.approver_chain_id = sqlc.arg(chain_id)::uuid AND g.chain_id IS NOT NULL
  UNION
  SELECT g.chain_id FROM login_grants g
  JOIN lineage l ON g.approver_chain_id = l.id
  WHERE g.chain_id IS NOT NULL
)
SELECT id::uuid FROM lineage;

-- name: DeleteExpiredLoginGrants :execrows
DELETE FROM login_grants g
WHERE g.expires_at < NOW() - interval '1 hour'
  AND (
    g.chain_id IS NULL
    OR g.approver_chain_id IS NULL
    OR NOT EXISTS (SELECT 1 FROM chains c WHERE c.id = g.chain_id)
  );
