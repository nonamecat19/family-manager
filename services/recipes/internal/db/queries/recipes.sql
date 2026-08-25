-- name: CreateRecipe :one
INSERT INTO recipes (family_id, title, description, category_id, subcategory_id,
    servings, prep_seconds, cook_seconds, author_user_id, notes, rating)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetRecipe :one
SELECT * FROM recipes
WHERE id = $1;

-- ListRecipes is the one filtered/sorted read behind the browse screen. Every filter is a
-- no-op sentinel when unset (NULL for the text/uuid ones, 0 for the numeric ones) so the app
-- sends one shape of request whether it is browsing a subcategory or searching the whole
-- cookbook. Sorting is a text discriminator rather than string-built SQL: the set of orders
-- is closed (see RecipeSort in the proto), so it belongs in the query, not in Go.
-- name: ListRecipes :many
SELECT r.* FROM recipes r
WHERE r.family_id = $1
  AND (sqlc.narg('category_id')::uuid IS NULL OR r.category_id = @category_id)
  AND (sqlc.narg('subcategory_id')::uuid IS NULL OR r.subcategory_id = @subcategory_id)
  AND (sqlc.narg('search')::text IS NULL
       OR lower(btrim(r.title)) LIKE '%' || lower(@search) || '%')
  AND (sqlc.narg('ingredient')::text IS NULL OR EXISTS (
        SELECT 1 FROM recipe_ingredients ri
        WHERE ri.recipe_id = r.id
          AND lower(btrim(ri.name)) LIKE '%' || lower(@ingredient) || '%'))
  AND r.rating >= sqlc.arg('min_rating')::int
  AND (sqlc.arg('max_total_seconds')::int = 0
       OR r.prep_seconds + r.cook_seconds <= sqlc.arg('max_total_seconds')::int)
  AND (NOT sqlc.arg('favorite_only')::bool OR EXISTS (
        SELECT 1 FROM recipe_favorites f
        WHERE f.recipe_id = r.id AND f.user_id = sqlc.arg('user_id')::uuid))
ORDER BY
    CASE WHEN sqlc.arg('sort')::text = 'title' THEN lower(btrim(r.title)) END ASC,
    CASE WHEN sqlc.arg('sort')::text = 'rating' THEN r.rating END DESC,
    CASE WHEN sqlc.arg('sort')::text = 'time' THEN r.prep_seconds + r.cook_seconds END ASC,
    CASE WHEN sqlc.arg('sort')::text = 'favorites' THEN r.favorite_count END DESC,
    r.created_at DESC;

-- name: ListFavoriteRecipes :many
SELECT r.* FROM recipes r
JOIN recipe_favorites f ON f.recipe_id = r.id
WHERE f.user_id = $1
ORDER BY f.created_at DESC;

-- name: UpdateRecipe :one
UPDATE recipes
SET title = $2, description = $3, category_id = $4, subcategory_id = $5,
    servings = $6, prep_seconds = $7, cook_seconds = $8, notes = $9, rating = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteRecipe :execrows
DELETE FROM recipes
WHERE id = $1;

-- name: UpdateRecipeImage :one
UPDATE recipes
SET image_url = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SetRecipeRating :one
UPDATE recipes
SET rating = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

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