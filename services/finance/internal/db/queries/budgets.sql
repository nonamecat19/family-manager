-- name: CreateBudget :one
INSERT INTO budgets (
    family_id, name, category_id, limit_minor, currency_code, period, start_on, sort_order
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListBudgets :many
SELECT * FROM budgets
WHERE family_id = @family_id
  AND (@include_archived::bool OR archived = FALSE)
ORDER BY sort_order, created_at;

-- name: GetBudget :one
SELECT * FROM budgets
WHERE id = @id AND family_id = @family_id;

-- name: UpdateBudget :one
UPDATE budgets
SET name          = @name,
    category_id   = @category_id,
    limit_minor   = @limit_minor,
    period        = @period,
    start_on      = @start_on,
    archived      = @archived,
    sort_order    = @sort_order,
    updated_at    = NOW()
WHERE id = @id AND family_id = @family_id
RETURNING *;

-- name: DeleteBudget :execrows
DELETE FROM budgets
WHERE id = @id AND family_id = @family_id;

-- Budgets are consumed by spending only: a transfer moves your own money between pockets and
-- income is not expenditure, so neither touches a limit.
--
-- A NULL category_id on the budget means "everything", which is why the category filter is
-- written as a nullable comparison rather than an equality.
--
-- name: SumBudgetSpend :one
SELECT COALESCE(SUM(amount_minor), 0)::bigint AS spent_minor
FROM transactions
WHERE family_id = @family_id
  AND type = 'expense'
  AND occurred_on >= @from_date
  AND occurred_on <= @to_date
  AND (@category_id::uuid IS NULL OR category_id = @category_id::uuid);

-- Budgets a given category's spending counts against: its own, plus the household total.
-- Used after a write to decide whether a limit was just crossed.
--
-- name: ListBudgetsForCategory :many
SELECT * FROM budgets
WHERE family_id = @family_id
  AND archived = FALSE
  AND (category_id IS NULL OR category_id = @category_id::uuid);
