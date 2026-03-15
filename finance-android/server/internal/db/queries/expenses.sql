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
