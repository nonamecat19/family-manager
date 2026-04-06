-- name: CreateInvestment :one
INSERT INTO investments (user_id, wallet_id, type, coin, amount, start_date, end_date, daily_reward, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetInvestmentByID :one
SELECT * FROM investments
WHERE id = $1;

-- name: GetInvestmentsByUserID :many
SELECT * FROM investments
WHERE user_id = $1
ORDER BY start_date DESC;

-- name: UpdateInvestment :one
UPDATE investments
SET wallet_id = $2, type = $3, coin = $4, amount = $5, start_date = $6, 
    end_date = $7, daily_reward = $8, status = $9
WHERE id = $1
RETURNING *;

-- name: DeleteInvestment :exec
DELETE FROM investments
WHERE id = $1;

