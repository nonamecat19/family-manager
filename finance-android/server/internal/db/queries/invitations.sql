-- name: CreateInvitation :one
INSERT INTO family_invitations (family_id, inviter_user_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetInvitationByTokenHash :one
SELECT fi.*, f.name as family_name
FROM family_invitations fi
JOIN families f ON f.id = fi.family_id
WHERE fi.token_hash = $1 AND fi.status = 'pending' AND fi.expires_at > NOW();

-- name: AcceptInvitation :execrows
UPDATE family_invitations SET status = 'accepted'
WHERE id = $1 AND status = 'pending';

-- name: RevokeInvitation :execrows
UPDATE family_invitations SET status = 'revoked'
WHERE id = $1 AND family_id = $2 AND status = 'pending';

-- name: GetPendingInvitations :many
SELECT id, family_id, inviter_user_id, status, expires_at, created_at
FROM family_invitations
WHERE family_id = $1 AND status = 'pending' AND expires_at > NOW()
ORDER BY created_at DESC;
