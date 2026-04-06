-- name: CreateRecurring :one
INSERT INTO recurring (user_id, wallet_id, type, amount, currency, description, recurrence, start_date, end_date, next_execution)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetRecurringByID :one
SELECT * FROM recurring
WHERE id = $1;

-- name: GetRecurringByUserID :many
SELECT * FROM recurring
WHERE user_id = $1
ORDER BY next_execution ASC;

-- name: GetDueRecurring :many
SELECT * FROM recurring
WHERE next_execution <= NOW()
  AND (end_date IS NULL OR end_date >= CURRENT_DATE)
ORDER BY next_execution ASC;

-- name: UpdateRecurring :one
UPDATE recurring
SET wallet_id = $2, type = $3, amount = $4, currency = $5, description = $6, 
    recurrence = $7, start_date = $8, end_date = $9, next_execution = $10
WHERE id = $1
RETURNING *;

-- name: UpdateRecurringNextExecution :one
UPDATE recurring
SET next_execution = $2
WHERE id = $1
RETURNING *;

-- name: DeleteRecurring :exec
DELETE FROM recurring
WHERE id = $1;

