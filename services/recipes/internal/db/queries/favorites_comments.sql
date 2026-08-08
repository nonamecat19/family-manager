-- name: AddFavorite :exec
INSERT INTO recipe_favorites (recipe_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM recipe_favorites
WHERE recipe_id = $1 AND user_id = $2;

-- name: IsFavorite :one
SELECT EXISTS (
    SELECT 1 FROM recipe_favorites WHERE recipe_id = $1 AND user_id = $2
) AS is_favorite;

-- name: IncrementFavoriteCount :exec
UPDATE recipes SET favorite_count = favorite_count + 1 WHERE id = $1;

-- name: DecrementFavoriteCount :exec
UPDATE recipes SET favorite_count = favorite_count - 1 WHERE id = $1;

-- name: AddComment :one
INSERT INTO recipe_comments (recipe_id, user_id, body)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListComments :many
SELECT * FROM recipe_comments
WHERE recipe_id = $1
ORDER BY created_at;