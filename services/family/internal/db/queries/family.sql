-- name: CreateFamily :one
INSERT INTO families (name, owner_user_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetFamily :one
SELECT * FROM families
WHERE id = $1;

-- name: UpdateFamily :one
UPDATE families
SET name = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteFamily :exec
DELETE FROM families
WHERE id = $1;

-- name: AddMember :one
INSERT INTO family_members (family_id, user_id, display_name, email, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListMembers :many
SELECT * FROM family_members
WHERE family_id = $1
ORDER BY joined_at;

-- name: GetMembership :one
SELECT * FROM family_members
WHERE user_id = $1
LIMIT 1;

-- name: GetMember :one
SELECT * FROM family_members
WHERE family_id = $1 AND user_id = $2;

-- name: RemoveMember :execrows
DELETE FROM family_members
WHERE family_id = $1 AND user_id = $2;

-- name: CountAdmins :one
SELECT count(*) FROM family_members
WHERE family_id = $1 AND role = 'admin';

-- name: CreateInvitation :one
INSERT INTO family_invitations (family_id, inviter_user_id, email, role, token_hash, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetInvitationByTokenHash :one
SELECT * FROM family_invitations
WHERE token_hash = $1;

-- name: GetInvitation :one
SELECT * FROM family_invitations
WHERE id = $1;

-- name: ListInvitations :many
SELECT * FROM family_invitations
WHERE family_id = $1
ORDER BY created_at DESC;

-- name: MarkInvitationAccepted :one
UPDATE family_invitations
SET status = 'accepted', accepted_by = $2
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: MarkInvitationRevoked :execrows
UPDATE family_invitations
SET status = 'revoked'
WHERE id = $1 AND status = 'pending';

-- name: ExpireStaleInvitations :execrows
UPDATE family_invitations
SET status = 'expired'
WHERE status = 'pending' AND expires_at < NOW();
