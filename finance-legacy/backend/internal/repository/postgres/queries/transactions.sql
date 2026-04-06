-- name: CreateTransaction :one
INSERT INTO transactions (wallet_id, type, amount, currency, description, category, executed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTransactionByID :one
SELECT * FROM transactions
WHERE id = $1;

-- name: GetTransactionsByWalletID :many
SELECT * FROM transactions
WHERE wallet_id = $1
  AND ($2::TIMESTAMP IS NULL OR executed_at >= $2)
  AND ($3::TIMESTAMP IS NULL OR executed_at <= $3)
ORDER BY executed_at DESC, created_at DESC;

-- name: UpdateTransaction :one
UPDATE transactions
SET type = $2, amount = $3, currency = $4, description = $5, category = $6, executed_at = $7
WHERE id = $1
RETURNING *;

-- name: DeleteTransaction :exec
DELETE FROM transactions
WHERE id = $1;

