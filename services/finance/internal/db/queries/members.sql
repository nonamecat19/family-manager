-- The member projection, maintained from family.member.* events. finance never writes it from
-- a client request: a member row appears because services/family said so.

-- name: ListMembers :many
SELECT * FROM finance_members
WHERE family_id = $1
  AND (sqlc.arg('include_pending')::bool OR status = 'active')
ORDER BY joined_at, user_id;

-- name: GetMember :one
SELECT * FROM finance_members
WHERE family_id = $1 AND user_id = $2;

-- name: UpsertMember :one
INSERT INTO finance_members (family_id, user_id, display_name, initial, avatar_color_step,
    role, status, email, joined_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
ON CONFLICT (family_id, user_id) DO UPDATE
SET display_name = EXCLUDED.display_name,
    initial = EXCLUDED.initial,
    avatar_color_step = EXCLUDED.avatar_color_step,
    role = EXCLUDED.role,
    status = EXCLUDED.status,
    email = EXCLUDED.email
RETURNING *;

-- name: DeleteMember :execrows
DELETE FROM finance_members
WHERE family_id = $1 AND user_id = $2;

-- name: CountMembers :one
SELECT COUNT(*) FROM finance_members
WHERE family_id = $1 AND status = 'active';
