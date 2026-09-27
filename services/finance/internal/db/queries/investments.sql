-- name: ListInvestments :many
SELECT i.*, c.group_id AS group_id,
    COALESCE((
        SELECT SUM(t.amount_minor)
        FROM transactions t
        JOIN accounts a ON a.id = t.account_id
        WHERE t.family_id = i.family_id
          AND t.category_id = i.category_id
          AND t.type = 'expense'
          AND t.currency_code = i.currency_code
          AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
    ), 0)::bigint AS invested_minor
FROM investments i
JOIN categories c ON c.id = i.category_id
WHERE i.family_id = $1
  AND (sqlc.arg('include_archived')::bool OR NOT i.archived)
ORDER BY i.sort_order, i.created_at;

-- name: GetInvestment :one
SELECT * FROM investments
WHERE id = $1 AND family_id = $2;

-- name: CreateInvestment :one
INSERT INTO investments (family_id, name, kind, currency_code, category_id, sort_order)
VALUES ($1, $2, $3, $4, $5,
    COALESCE((SELECT MAX(sort_order) + 1 FROM investments WHERE family_id = $1), 0))
RETURNING *;

-- name: UpdateInvestment :one
UPDATE investments
SET name       = COALESCE(sqlc.narg('name')::text, name),
    kind       = COALESCE(sqlc.narg('kind')::text, kind),
    archived   = COALESCE(sqlc.narg('archived')::bool, archived),
    updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: SetInvestmentValue :one
UPDATE investments
SET current_value_minor = $3, value_updated_on = $4, updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: DeleteInvestment :execrows
DELETE FROM investments
WHERE id = $1 AND family_id = $2;
