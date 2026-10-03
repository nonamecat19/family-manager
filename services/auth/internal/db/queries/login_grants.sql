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
SET user_id = sqlc.arg(user_id), approved_at = sqlc.arg(decided_at)
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
UPDATE login_grants
SET consumed_at = NOW()
WHERE id = $1
  AND approved_at IS NOT NULL
  AND denied_at IS NULL
  AND consumed_at IS NULL;

-- name: DeleteExpiredLoginGrants :execrows
DELETE FROM login_grants
WHERE expires_at < NOW() - interval '1 hour';
