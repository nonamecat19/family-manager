-- name: CreateFamily :one
INSERT INTO families (name, admin_user_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetFamilyByUserID :one
SELECT f.*
FROM families f
JOIN family_members fm ON fm.family_id = f.id
WHERE fm.user_id = $1;

-- name: GetFamilyMembers :many
SELECT fm.id, fm.user_id, u.email, fm.role, fm.joined_at
FROM family_members fm
JOIN users u ON u.id = fm.user_id
WHERE fm.family_id = $1
ORDER BY fm.joined_at;

-- name: AddFamilyMember :one
INSERT INTO family_members (family_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: RemoveFamilyMember :execrows
DELETE FROM family_members
WHERE family_id = $1 AND user_id = $2 AND role != 'admin';

-- name: DeleteFamily :execrows
DELETE FROM families
WHERE id = $1 AND admin_user_id = $2;

-- name: GetFamilyMemberCount :one
SELECT COUNT(*) FROM family_members WHERE family_id = $1;
