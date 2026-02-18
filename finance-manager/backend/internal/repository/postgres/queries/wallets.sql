-- name: CreateWallet :one
INSERT INTO wallets (user_id, name, type, currency, balance, category)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, name, type, currency, balance, category, created_at;

-- name: GetWalletByID :one
SELECT id, user_id, name, type, currency, balance, category, created_at FROM wallets
WHERE id = $1;

-- name: GetWalletsByUserID :many
SELECT id, user_id, name, type, currency, balance, category, created_at FROM wallets
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateWallet :one
UPDATE wallets
SET name = COALESCE($2, name), 
    type = COALESCE($3, type), 
    currency = COALESCE($4, currency)
WHERE id = $1
RETURNING id, user_id, name, type, currency, balance, category, created_at;

-- name: UpdateWalletCategory :one
UPDATE wallets
SET category = $2
WHERE id = $1
RETURNING id, user_id, name, type, currency, balance, category, created_at;

-- name: UpdateWalletBalance :one
UPDATE wallets
SET balance = $2
WHERE id = $1
RETURNING id, user_id, name, type, currency, balance, category, created_at;

-- name: DeleteWallet :exec
DELETE FROM wallets
WHERE id = $1;

