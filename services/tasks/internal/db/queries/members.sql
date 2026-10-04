-- name: TouchKnownMember :exec
INSERT INTO known_members (family_id, user_id, email)
VALUES ($1, $2, $3)
ON CONFLICT (family_id, user_id) DO UPDATE
SET email = CASE WHEN EXCLUDED.email <> '' THEN EXCLUDED.email ELSE known_members.email END,
    seen_at = NOW();

-- name: UpsertKnownMember :exec
INSERT INTO known_members (family_id, user_id, display_name, email)
VALUES ($1, $2, $3, $4)
ON CONFLICT (family_id, user_id) DO UPDATE
SET display_name = EXCLUDED.display_name, email = EXCLUDED.email, seen_at = NOW();

-- name: DeleteKnownMembersExcept :exec
DELETE FROM known_members
WHERE family_id = $1 AND cardinality(COALESCE(sqlc.arg(keep)::uuid[], '{}')) > 0
  AND NOT (user_id = ANY(sqlc.arg(keep)::uuid[]));

-- name: DeleteKnownMember :exec
DELETE FROM known_members
WHERE family_id = $1 AND user_id = $2;

-- name: ListKnownMembers :many
SELECT * FROM known_members
WHERE family_id = $1
ORDER BY user_id;
