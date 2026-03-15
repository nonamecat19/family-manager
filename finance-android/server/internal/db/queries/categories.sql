-- name: CreateCategory :one
INSERT INTO categories (user_id, name, icon, color, sort_order)
VALUES ($1, $2, $3, $4, (SELECT COALESCE(MAX(sort_order), -1) + 1 FROM categories WHERE user_id = $1))
RETURNING id, user_id, name, icon, color, sort_order, created_at, updated_at;

-- name: GetCategoriesByUser :many
SELECT id, user_id, name, icon, color, sort_order, created_at, updated_at
FROM categories
WHERE user_id = $1
ORDER BY sort_order ASC;

-- name: GetCategoryByID :one
SELECT id, user_id, name, icon, color, sort_order, created_at, updated_at
FROM categories
WHERE id = $1 AND user_id = $2;

-- name: UpdateCategory :execrows
UPDATE categories
SET name = $3, icon = $4, color = $5, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: DeleteCategory :execrows
DELETE FROM categories
WHERE id = $1 AND user_id = $2;

-- name: UpdateCategorySortOrder :exec
UPDATE categories
SET sort_order = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $2;
