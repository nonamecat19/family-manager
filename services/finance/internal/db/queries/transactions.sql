-- name: CreateTransaction :one
INSERT INTO transactions (family_id, type, account_id, counter_account_id, category_id,
    amount_minor, currency_code, received_amount_minor, received_currency_code,
    note, merchant, occurred_on, member_id, created_by_user_id, template_id, recurring_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: GetVisibleTransaction :one
SELECT t.*, c.group_id AS group_id
FROM transactions t
JOIN accounts a ON a.id = t.account_id
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.id = $1
  AND t.family_id = $2
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid);

-- name: UpdateTransaction :one
UPDATE transactions
SET type        = COALESCE(sqlc.narg('type')::text, type),
    account_id  = COALESCE(sqlc.narg('account_id')::uuid, account_id),
    category_id = COALESCE(sqlc.narg('category_id')::uuid, category_id),
    amount_minor = COALESCE(sqlc.narg('amount_minor')::bigint, amount_minor),
    currency_code = COALESCE(sqlc.narg('currency_code')::text, currency_code),
    note        = COALESCE(sqlc.narg('note')::text, note),
    merchant    = COALESCE(sqlc.narg('merchant')::text, merchant),
    occurred_on = COALESCE(sqlc.narg('occurred_on')::date, occurred_on),
    member_id   = COALESCE(sqlc.narg('member_id')::uuid, member_id),
    updated_at  = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: DeleteTransaction :execrows
DELETE FROM transactions
WHERE id = $1 AND family_id = $2;

-- name: ListVisibleTransactions :many
SELECT t.*, c.group_id AS group_id
FROM transactions t
JOIN accounts a ON a.id = t.account_id
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND t.occurred_on >= sqlc.arg('from_date')::date
  AND t.occurred_on <= sqlc.arg('to_date')::date
  AND (sqlc.narg('kind')::text IS NULL OR t.type = @kind)
  AND (sqlc.arg('include_transfers')::bool OR t.type <> 'transfer')
  AND (cardinality(sqlc.arg('member_ids')::uuid[]) = 0 OR t.member_id = ANY(@member_ids::uuid[]))
  AND (cardinality(sqlc.arg('account_ids')::uuid[]) = 0 OR t.account_id = ANY(@account_ids::uuid[]))
  AND (cardinality(sqlc.arg('category_ids')::uuid[]) = 0 OR t.category_id = ANY(@category_ids::uuid[]))
  AND (cardinality(sqlc.arg('group_ids')::uuid[]) = 0 OR c.group_id = ANY(@group_ids::uuid[]))
  AND (sqlc.narg('query')::text IS NULL
       OR lower(t.note) LIKE '%' || replace(replace(replace(lower(@query), '\', '\\'), '%', '\%'), '_', '\_') || '%' ESCAPE '\'
       OR lower(t.merchant) LIKE '%' || replace(replace(replace(lower(@query), '\', '\\'), '%', '\%'), '_', '\_') || '%' ESCAPE '\')
  AND (sqlc.narg('cursor_date')::date IS NULL
       OR t.occurred_on < @cursor_date
       OR (t.occurred_on = @cursor_date AND t.id < sqlc.narg('cursor_id')::uuid))
ORDER BY t.occurred_on DESC, t.id DESC
LIMIT sqlc.arg('page_size')::int;

-- name: SumVisibleTransactions :one
SELECT COALESCE(SUM(CASE WHEN sqlc.narg('kind')::text IS NULL AND t.type = 'income'
                         THEN -t.amount_minor ELSE t.amount_minor END), 0)::bigint AS total_minor,
       COUNT(*)::int AS transaction_count
FROM transactions t
JOIN accounts a ON a.id = t.account_id
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND t.occurred_on >= sqlc.arg('from_date')::date
  AND t.occurred_on <= sqlc.arg('to_date')::date
  AND t.type <> 'transfer'
  AND t.currency_code = sqlc.arg('currency_code')::text
  AND (sqlc.narg('kind')::text IS NULL OR t.type = @kind)
  AND (cardinality(sqlc.arg('member_ids')::uuid[]) = 0 OR t.member_id = ANY(@member_ids::uuid[]))
  AND (cardinality(sqlc.arg('account_ids')::uuid[]) = 0 OR t.account_id = ANY(@account_ids::uuid[]))
  AND (cardinality(sqlc.arg('category_ids')::uuid[]) = 0 OR t.category_id = ANY(@category_ids::uuid[]))
  AND (cardinality(sqlc.arg('group_ids')::uuid[]) = 0 OR c.group_id = ANY(@group_ids::uuid[]))
  AND (sqlc.narg('query')::text IS NULL
       OR lower(t.note) LIKE '%' || replace(replace(replace(lower(@query), '\', '\\'), '%', '\%'), '_', '\_') || '%' ESCAPE '\'
       OR lower(t.merchant) LIKE '%' || replace(replace(replace(lower(@query), '\', '\\'), '%', '\%'), '_', '\_') || '%' ESCAPE '\');

-- name: SumByGroup :many
SELECT c.group_id, COALESCE(SUM(CASE WHEN sqlc.narg('kind')::text IS NULL AND t.type = 'income'
                         THEN -t.amount_minor ELSE t.amount_minor END), 0)::bigint AS total_minor,
       COUNT(*)::int AS transaction_count
FROM transactions t
JOIN accounts a ON a.id = t.account_id
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND t.occurred_on >= sqlc.arg('from_date')::date
  AND t.occurred_on <= sqlc.arg('to_date')::date
  AND t.type <> 'transfer'
  AND t.currency_code = sqlc.arg('currency_code')::text
  AND (sqlc.narg('kind')::text IS NULL OR t.type = @kind)
  AND (cardinality(sqlc.arg('member_ids')::uuid[]) = 0 OR t.member_id = ANY(@member_ids::uuid[]))
  AND (cardinality(sqlc.arg('account_ids')::uuid[]) = 0 OR t.account_id = ANY(@account_ids::uuid[]))
GROUP BY c.group_id;

-- name: SumByCategory :many
SELECT t.category_id, COALESCE(SUM(CASE WHEN sqlc.narg('kind')::text IS NULL AND t.type = 'income'
                         THEN -t.amount_minor ELSE t.amount_minor END), 0)::bigint AS total_minor,
       COUNT(*)::int AS transaction_count
FROM transactions t
JOIN accounts a ON a.id = t.account_id
JOIN categories c ON c.id = t.category_id
WHERE t.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND t.occurred_on >= sqlc.arg('from_date')::date
  AND t.occurred_on <= sqlc.arg('to_date')::date
  AND t.type <> 'transfer'
  AND t.currency_code = sqlc.arg('currency_code')::text
  AND (sqlc.narg('kind')::text IS NULL OR t.type = @kind)
  AND (sqlc.narg('group_id')::uuid IS NULL OR c.group_id = @group_id)
  AND (cardinality(sqlc.arg('member_ids')::uuid[]) = 0 OR t.member_id = ANY(@member_ids::uuid[]))
GROUP BY t.category_id;

-- name: SumByMember :many
SELECT t.member_id, c.group_id, COALESCE(SUM(CASE WHEN sqlc.narg('kind')::text IS NULL AND t.type = 'income'
                         THEN -t.amount_minor ELSE t.amount_minor END), 0)::bigint AS total_minor,
       COUNT(*)::int AS transaction_count
FROM transactions t
JOIN accounts a ON a.id = t.account_id
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND t.occurred_on >= sqlc.arg('from_date')::date
  AND t.occurred_on <= sqlc.arg('to_date')::date
  AND t.type <> 'transfer'
  AND t.currency_code = sqlc.arg('currency_code')::text
  AND (sqlc.narg('kind')::text IS NULL OR t.type = @kind)
  AND (cardinality(sqlc.arg('account_ids')::uuid[]) = 0 OR t.account_id = ANY(@account_ids::uuid[]))
GROUP BY t.member_id, c.group_id;

-- name: SumDailyTotals :many
SELECT t.occurred_on, t.member_id, c.group_id,
       COALESCE(SUM(CASE WHEN sqlc.narg('kind')::text IS NULL AND t.type = 'income'
                         THEN -t.amount_minor ELSE t.amount_minor END), 0)::bigint AS total_minor
FROM transactions t
JOIN accounts a ON a.id = t.account_id
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND t.occurred_on >= sqlc.arg('from_date')::date
  AND t.occurred_on <= sqlc.arg('to_date')::date
  AND t.type <> 'transfer'
  AND t.currency_code = sqlc.arg('currency_code')::text
  AND (sqlc.narg('kind')::text IS NULL OR t.type = @kind)
  AND (cardinality(sqlc.arg('member_ids')::uuid[]) = 0 OR t.member_id = ANY(@member_ids::uuid[]))
  AND (cardinality(sqlc.arg('account_ids')::uuid[]) = 0 OR t.account_id = ANY(@account_ids::uuid[]))
GROUP BY t.occurred_on, t.member_id, c.group_id
ORDER BY t.occurred_on;

-- name: SumBudgetSpend :one
SELECT COALESCE(SUM(t.amount_minor), 0)::bigint AS total_minor
FROM transactions t
JOIN accounts a ON a.id = t.account_id
JOIN categories c ON c.id = t.category_id
WHERE t.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND t.occurred_on >= sqlc.arg('from_date')::date
  AND t.occurred_on <= sqlc.arg('to_date')::date
  AND t.type = 'expense'
  AND t.currency_code = sqlc.arg('currency_code')::text
  AND (sqlc.narg('group_id')::uuid IS NULL OR c.group_id = @group_id)
  AND (sqlc.narg('category_id')::uuid IS NULL OR t.category_id = @category_id)
  AND (sqlc.narg('member_id')::uuid IS NULL OR t.member_id = @member_id);

-- name: CountTransactionsForRecurringOccurrence :one
SELECT COUNT(*) FROM transactions
WHERE family_id = $1 AND recurring_id = $2 AND occurred_on = $3;
