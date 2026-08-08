-- name: CreateCategory :one
INSERT INTO recipe_categories (family_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: ListCategories :many
SELECT * FROM recipe_categories
WHERE family_id = $1
ORDER BY name;

-- name: CreateSubcategory :one
INSERT INTO recipe_subcategories (category_id, family_id, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSubcategories :many
SELECT * FROM recipe_subcategories
WHERE category_id = $1
ORDER BY name;

-- name: GetSubcategory :one
SELECT * FROM recipe_subcategories
WHERE id = $1;