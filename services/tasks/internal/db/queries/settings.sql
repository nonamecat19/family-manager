-- name: InitFamilySettings :exec
INSERT INTO family_settings (family_id, timezone)
VALUES ($1, $2)
ON CONFLICT (family_id) DO NOTHING;

-- name: GetFamilySettings :one
SELECT * FROM family_settings
WHERE family_id = $1;

-- name: SetFamilyTimezone :one
INSERT INTO family_settings (family_id, timezone)
VALUES ($1, $2)
ON CONFLICT (family_id) DO UPDATE
SET timezone = EXCLUDED.timezone, updated_at = NOW()
RETURNING *;

-- name: ListAllFamilySettings :many
SELECT * FROM family_settings
ORDER BY family_id;
