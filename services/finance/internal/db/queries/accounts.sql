-- name: ListVisibleAccounts :many
SELECT a.*,
    (a.opening_balance_minor
    + COALESCE((SELECT SUM(CASE t.type WHEN 'income' THEN t.amount_minor ELSE -t.amount_minor END)
                FROM transactions t WHERE t.account_id = a.id), 0)
    + COALESCE((SELECT SUM(COALESCE(t.received_amount_minor, t.amount_minor))
                FROM transactions t WHERE t.counter_account_id = a.id), 0)
    )::bigint AS balance_minor
FROM accounts a
WHERE a.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND (sqlc.arg('include_archived')::bool OR NOT a.archived)
ORDER BY a.sort_order, a.created_at;

-- name: CountHiddenPrivateAccounts :many
SELECT a.owner_member_id, COUNT(*)::int AS account_count
FROM accounts a
WHERE a.family_id = $1
  AND a.visibility = 'private'
  AND a.owner_member_id IS DISTINCT FROM sqlc.arg('viewer_member_id')::uuid
  AND NOT a.archived
GROUP BY a.owner_member_id
ORDER BY a.owner_member_id;

-- name: GetVisibleAccount :one
SELECT a.*,
    (a.opening_balance_minor
    + COALESCE((SELECT SUM(CASE t.type WHEN 'income' THEN t.amount_minor ELSE -t.amount_minor END)
                FROM transactions t WHERE t.account_id = a.id), 0)
    + COALESCE((SELECT SUM(COALESCE(t.received_amount_minor, t.amount_minor))
                FROM transactions t WHERE t.counter_account_id = a.id), 0)
    )::bigint AS balance_minor
FROM accounts a
WHERE a.id = $1
  AND a.family_id = $2
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid);

-- name: SumFamilyBalances :one
SELECT
    COALESCE(SUM(b.balance_minor) FILTER (
        WHERE b.kind <> 'savings' AND NOT b.excluded_from_family_total), 0)::bigint AS shared_balance_minor,
    COALESCE(SUM(b.balance_minor) FILTER (
        WHERE b.kind = 'savings' AND NOT b.excluded_from_family_total), 0)::bigint AS savings_minor,
    COUNT(*)::int AS shared_account_count
FROM (
    SELECT a.kind, a.excluded_from_family_total,
        (a.opening_balance_minor
        + COALESCE((SELECT SUM(CASE t.type WHEN 'income' THEN t.amount_minor ELSE -t.amount_minor END)
                    FROM transactions t WHERE t.account_id = a.id), 0)
        + COALESCE((SELECT SUM(COALESCE(t.received_amount_minor, t.amount_minor))
                    FROM transactions t WHERE t.counter_account_id = a.id), 0)
        )::bigint AS balance_minor
    FROM accounts a
    WHERE a.family_id = $1
      AND a.visibility = 'shared'
      AND NOT a.archived
      AND a.currency_code = sqlc.arg('currency_code')::text
) b;

-- name: CreateAccount :one
INSERT INTO accounts (family_id, name, kind, visibility, owner_member_id, currency_code,
    opening_balance_minor, icon, color_step, excluded_from_family_total, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    COALESCE((SELECT MAX(sort_order) + 1 FROM accounts WHERE family_id = $1), 0))
RETURNING *;

-- name: UpdateAccount :one
UPDATE accounts
SET name       = COALESCE(sqlc.narg('name')::text, name),
    kind       = COALESCE(sqlc.narg('kind')::text, kind),
    icon       = COALESCE(sqlc.narg('icon')::text, icon),
    color_step = COALESCE(sqlc.narg('color_step')::int, color_step),
    excluded_from_family_total =
        COALESCE(sqlc.narg('excluded_from_family_total')::bool, excluded_from_family_total),
    opening_balance_minor = COALESCE(sqlc.narg('opening_balance_minor')::bigint, opening_balance_minor),
    updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: SetAccountVisibility :one
UPDATE accounts
SET visibility = $3,
    owner_member_id = sqlc.narg('owner_member_id')::uuid,
    excluded_from_family_total = ($3 = 'private') OR excluded_from_family_total,
    updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: SetAccountArchived :one
UPDATE accounts
SET archived = $3, updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: DeleteAccount :execrows
DELETE FROM accounts
WHERE id = $1 AND family_id = $2;

-- name: ReorderAccount :exec
UPDATE accounts
SET sort_order = $3, updated_at = NOW()
WHERE id = $1 AND family_id = $2
  AND (visibility = 'shared' OR owner_member_id = sqlc.arg('viewer_member_id')::uuid);

-- name: CountAccountTransactions :one
SELECT COUNT(*) FROM transactions
WHERE family_id = $2 AND (account_id = $1 OR counter_account_id = $1);
