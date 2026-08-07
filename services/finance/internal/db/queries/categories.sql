-- name: CreateCategory :one
INSERT INTO categories (family_id, name, kind, color, icon, parent_id, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListCategories :many
SELECT * FROM categories
WHERE family_id = @family_id
  AND (@kind::text = '' OR kind = @kind::text)
  AND (@include_archived::bool OR archived = FALSE)
ORDER BY sort_order, name;

-- name: GetCategory :one
SELECT * FROM categories
WHERE id = @id AND family_id = @family_id;

-- name: UpdateCategory :one
UPDATE categories
SET name       = @name,
    color      = @color,
    icon       = @icon,
    parent_id  = @parent_id,
    archived   = @archived,
    sort_order = @sort_order,
    updated_at = NOW()
WHERE id = @id AND family_id = @family_id
RETURNING *;

-- name: DeleteCategory :execrows
DELETE FROM categories
WHERE id = @id AND family_id = @family_id;

-- name: CountCategoryTransactions :one
SELECT count(*) FROM transactions
WHERE category_id = @category_id;
