-- name: UpsertTag :one
INSERT INTO tags (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: GetAllTags :many
SELECT * FROM tags ORDER BY name;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = $1;

-- name: GetTagByID :one
SELECT * FROM tags WHERE id = $1;

-- name: AddTagToList :exec
INSERT INTO list_tags (list_id, tag_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromList :exec
DELETE FROM list_tags WHERE list_id = $1 AND tag_id = $2;

-- name: GetTagsByList :many
SELECT t.* FROM tags t
JOIN list_tags lt ON lt.tag_id = t.id
WHERE lt.list_id = $1
ORDER BY t.name;

-- name: GetListsByTag :many
SELECT l.* FROM lists l
JOIN list_tags lt ON lt.list_id = l.id
WHERE lt.tag_id = $1
ORDER BY l.created_at DESC;
