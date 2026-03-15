-- name: CreateExpense :one
INSERT INTO expenses (user_id, category_id, amount_cents, note, expense_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, category_id, amount_cents, note, expense_date, created_at, updated_at;

-- name: GetExpensesByUser :many
SELECT id, user_id, category_id, amount_cents, note, expense_date, created_at, updated_at
FROM expenses
WHERE user_id = $1
ORDER BY expense_date DESC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateExpense :one
UPDATE expenses
SET category_id = $3, amount_cents = $4, note = $5, expense_date = $6, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, category_id, amount_cents, note, expense_date, created_at, updated_at;

-- name: DeleteExpense :execrows
DELETE FROM expenses
WHERE id = $1 AND user_id = $2;
