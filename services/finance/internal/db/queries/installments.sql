-- name: ListVisibleInstallments :many
SELECT sqlc.embed(i), c.group_id AS group_id,
    COALESCE(p.paid_minor, 0)::bigint AS paid_minor,
    COALESCE(p.payments, 0)::bigint AS payments
FROM installments i
JOIN accounts a ON a.id = i.account_id
JOIN categories c ON c.id = i.category_id
LEFT JOIN LATERAL (
    SELECT SUM(t.amount_minor) AS paid_minor, COUNT(*) AS payments
    FROM transactions t
    WHERE t.family_id = i.family_id
      AND t.category_id = i.category_id
      AND t.type = 'expense'
      AND t.currency_code = i.currency_code
) p ON TRUE
WHERE i.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND (sqlc.arg('include_closed')::bool OR i.status = 'active')
ORDER BY i.status = 'active' DESC, i.next_due_on, i.created_at;

-- name: GetVisibleInstallment :one
SELECT i.*
FROM installments i
JOIN accounts a ON a.id = i.account_id
WHERE i.id = $1
  AND i.family_id = $2
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid);

-- name: CreateInstallment :one
INSERT INTO installments (family_id, name, total_minor, monthly_minor, months, currency_code,
    account_id, category_id, member_id, created_by_user_id, purchased_on, day_of_month, next_due_on)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: UpdateInstallment :one
UPDATE installments
SET name          = COALESCE(sqlc.narg('name')::text, name),
    monthly_minor = COALESCE(sqlc.narg('monthly_minor')::bigint, monthly_minor),
    account_id    = COALESCE(sqlc.narg('account_id')::uuid, account_id),
    member_id     = COALESCE(sqlc.narg('member_id')::uuid, member_id),
    day_of_month  = COALESCE(sqlc.narg('day_of_month')::int, day_of_month),
    next_due_on   = COALESCE(sqlc.narg('next_due_on')::date, next_due_on),
    status        = COALESCE(sqlc.narg('status')::text, status),
    updated_at    = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: DeleteInstallment :execrows
DELETE FROM installments
WHERE id = $1 AND family_id = $2;

-- name: ListDueInstallments :many
SELECT id, family_id, category_id, next_due_on
FROM installments
WHERE status = 'active'
  AND next_due_on <= sqlc.arg('latest_due_on')::date
ORDER BY next_due_on
LIMIT sqlc.arg('max_rows')::int;

-- name: LockDueInstallment :one
SELECT * FROM installments
WHERE id = $1 AND status = 'active'
FOR UPDATE SKIP LOCKED;

-- name: SumInstallmentPayments :one
SELECT COALESCE(SUM(amount_minor), 0)::bigint AS paid_minor, COUNT(*)::bigint AS payments
FROM transactions
WHERE family_id = $1 AND category_id = $2 AND type = 'expense' AND currency_code = $3;

-- name: AdvanceInstallment :one
UPDATE installments
SET next_due_on = sqlc.arg('next_due_on')::date,
    status      = sqlc.arg('status')::text,
    updated_at  = NOW()
WHERE id = $1 AND next_due_on = sqlc.arg('expected_due_on')::date AND status = 'active'
RETURNING *;

-- name: GetInstallmentByID :one
SELECT * FROM installments
WHERE id = $1;
