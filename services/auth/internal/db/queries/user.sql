-- name: CreateUser :one
INSERT INTO users (email, name, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- Lookups are case-insensitive to match idx_users_email_lower: the address a user typed with
-- a capital must find the account they registered without one.
--
-- name: GetUserByEmail :one
SELECT * FROM users
WHERE lower(email) = lower($1)
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;
