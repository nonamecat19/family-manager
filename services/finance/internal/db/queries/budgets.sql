-- name: ListBudgets :many
SELECT * FROM budgets
WHERE family_id = $1
  AND (sqlc.narg('target_kind')::text IS NULL OR target_kind = @target_kind)
  AND (sqlc.arg('include_archived')::bool OR NOT archived)
ORDER BY sort_order, created_at;

-- name: GetBudget :one
SELECT * FROM budgets
WHERE id = $1 AND family_id = $2;

-- ListBudgetsForCategory is what CreateTransaction/UpdateTransaction/DeleteTransaction use to
-- answer affected_budgets: both the category's own budget and its group's, because spend in a
-- category counts toward both.
-- name: ListBudgetsForCategory :many
SELECT b.* FROM budgets b
LEFT JOIN categories c ON c.id = sqlc.narg('category_id')::uuid
WHERE b.family_id = $1
  AND NOT b.archived
  AND ((b.target_kind = 'category' AND b.category_id = sqlc.narg('category_id')::uuid)
       OR (b.target_kind = 'group' AND b.group_id = c.group_id))
ORDER BY b.sort_order;

-- name: CreateBudget :one
INSERT INTO budgets (family_id, target_kind, group_id, category_id, limit_minor,
    currency_code, period, start_on, member_id, notify_on_exceed, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    COALESCE((SELECT MAX(sort_order) + 1 FROM budgets WHERE family_id = $1), 0))
RETURNING *;

-- name: UpdateBudget :one
UPDATE budgets
SET limit_minor      = COALESCE(sqlc.narg('limit_minor')::bigint, limit_minor),
    period           = COALESCE(sqlc.narg('period')::text, period),
    start_on         = COALESCE(sqlc.narg('start_on')::date, start_on),
    notify_on_exceed = COALESCE(sqlc.narg('notify_on_exceed')::bool, notify_on_exceed),
    archived         = COALESCE(sqlc.narg('archived')::bool, archived),
    updated_at       = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: DeleteBudget :execrows
DELETE FROM budgets
WHERE id = $1 AND family_id = $2;
