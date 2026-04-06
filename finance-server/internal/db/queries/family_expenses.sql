-- name: GetFamilyExpenses :many
SELECT
    e.id,
    e.user_id,
    u.email AS user_email,
    e.category_id,
    c.name AS category_name,
    c.color AS category_color,
    c.icon AS category_icon,
    e.amount_cents,
    e.note,
    e.expense_date,
    e.created_at
FROM expenses e
JOIN family_members fm ON fm.user_id = e.user_id
JOIN users u ON u.id = e.user_id
JOIN categories c ON c.id = e.category_id
WHERE fm.family_id = $1
ORDER BY e.expense_date DESC, e.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetFamilyMemberTotals :many
SELECT
    e.user_id,
    u.email AS user_email,
    SUM(e.amount_cents)::BIGINT AS total_cents,
    COUNT(*)::INT AS expense_count
FROM expenses e
JOIN family_members fm ON fm.user_id = e.user_id
JOIN users u ON u.id = e.user_id
WHERE fm.family_id = $1
  AND e.expense_date >= $2
  AND e.expense_date <= $3
GROUP BY e.user_id, u.email
ORDER BY total_cents DESC;

-- name: GetFamilyCategoryTotals :many
SELECT
    e.category_id,
    c.name AS category_name,
    c.color AS category_color,
    c.icon AS category_icon,
    SUM(e.amount_cents)::BIGINT AS total_cents,
    COUNT(*)::INT AS expense_count
FROM expenses e
JOIN family_members fm ON fm.user_id = e.user_id
JOIN categories c ON c.id = e.category_id
WHERE fm.family_id = $1
  AND e.expense_date >= $2
  AND e.expense_date <= $3
GROUP BY e.category_id, c.name, c.color, c.icon
ORDER BY total_cents DESC;
