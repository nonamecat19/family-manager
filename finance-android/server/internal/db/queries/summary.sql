-- name: GetCategoryTotals :many
SELECT
    e.category_id,
    c.name AS category_name,
    c.color AS category_color,
    c.icon AS category_icon,
    SUM(e.amount_cents)::BIGINT AS total_cents,
    COUNT(*)::INT AS expense_count
FROM expenses e
JOIN categories c ON c.id = e.category_id
WHERE e.user_id = $1
  AND e.expense_date >= $2
  AND e.expense_date <= $3
GROUP BY e.category_id, c.name, c.color, c.icon
ORDER BY total_cents DESC;

-- name: GetDailyTotals :many
SELECT
    expense_date AS date,
    SUM(amount_cents)::BIGINT AS total_cents
FROM expenses
WHERE user_id = $1
  AND expense_date >= $2
  AND expense_date <= $3
GROUP BY expense_date
ORDER BY expense_date ASC;
