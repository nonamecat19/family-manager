-- name: CreateRecipe :one
INSERT INTO recipes (family_id, title, description, category_id, subcategory_id,
    servings, prep_seconds, cook_seconds, author_user_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetRecipe :one
SELECT * FROM recipes
WHERE id = $1;

-- name: ListRecipes :many
SELECT * FROM recipes
WHERE family_id = $1
  AND (sqlc.narg('category_id')::uuid IS NULL OR category_id = @category_id)
  AND (sqlc.narg('subcategory_id')::uuid IS NULL OR subcategory_id = @subcategory_id)
  AND (sqlc.narg('search')::text IS NULL OR lower(btrim(title)) LIKE '%' || lower(@search) || '%')
ORDER BY created_at DESC;

-- name: ListFavoriteRecipes :many
SELECT r.* FROM recipes r
JOIN recipe_favorites f ON f.recipe_id = r.id
WHERE f.user_id = $1
ORDER BY f.created_at DESC;

-- name: UpdateRecipe :one
UPDATE recipes
SET title = $2, description = $3, category_id = $4, subcategory_id = $5,
    servings = $6, prep_seconds = $7, cook_seconds = $8, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteRecipe :execrows
DELETE FROM recipes
WHERE id = $1;

-- name: IncrementCommentCount :exec
UPDATE recipes SET comment_count = comment_count + 1 WHERE id = $1;

-- name: AddIngredient :exec
INSERT INTO recipe_ingredients (recipe_id, position, name, amount, unit)
VALUES ($1, $2, $3, $4, $5);

-- name: AddStep :exec
INSERT INTO recipe_steps (recipe_id, position, instruction, duration_seconds)
VALUES ($1, $2, $3, $4);

-- name: ListIngredients :many
SELECT * FROM recipe_ingredients
WHERE recipe_id = $1
ORDER BY position;

-- name: ListSteps :many
SELECT * FROM recipe_steps
WHERE recipe_id = $1
ORDER BY position;

-- name: DeleteIngredients :exec
DELETE FROM recipe_ingredients WHERE recipe_id = $1;

-- name: DeleteSteps :exec
DELETE FROM recipe_steps WHERE recipe_id = $1;