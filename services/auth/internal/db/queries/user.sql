-- name: CreateUser :one
INSERT INTO users (email, name, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE lower(email) = lower($1)
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;
