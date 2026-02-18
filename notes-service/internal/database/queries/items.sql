-- name: CreateItem :one
INSERT INTO items (list_id, type, content, position)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetItemsByList :many
SELECT * FROM items WHERE list_id = $1 ORDER BY position ASC;

-- name: GetItemByID :one
SELECT * FROM items WHERE id = $1;

-- name: UpdateItem :one
UPDATE items
SET content = $2, type = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateItemPosition :exec
UPDATE items SET position = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteItem :exec
DELETE FROM items WHERE id = $1;
