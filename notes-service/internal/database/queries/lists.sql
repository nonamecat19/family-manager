-- name: CreateList :one
INSERT INTO lists (owner_id, title, description, is_public)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetListByID :one
SELECT * FROM lists WHERE id = $1;

-- name: GetListsByOwner :many
SELECT * FROM lists WHERE owner_id = $1 ORDER BY created_at DESC;

-- name: GetPublicLists :many
SELECT * FROM lists WHERE is_public = true ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateList :one
UPDATE lists
SET title = $2, description = $3, is_public = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteList :exec
DELETE FROM lists WHERE id = $1;
