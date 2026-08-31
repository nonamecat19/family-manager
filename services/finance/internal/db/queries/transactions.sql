-- The visibility boundary again, this time on the ledger: a transaction is readable when the
-- account it was paid from is readable. Every read in this file joins accounts and carries
-- @viewer_member_id, so "shared accounts plus my own private ones" is one predicate written
-- once rather than a filter each handler could forget.
--
-- Aggregates therefore include the caller's own private spend and no one else's — the design
-- excludes private BALANCES from the family headline (SumFamilyBalances does that), not the
-- caller's own spending from their own donut.
--
-- Transfers are excluded from every income/expense total: moving money between two of your
-- own accounts is not spending, and a report that counted it would double the month.

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

-- ListVisibleTransactions is the feed. Every filter is a no-op sentinel when unset — NULL for
-- the text and uuid ones, an empty array for the repeated ones — so the app sends one shape of
-- request whether it is browsing a month or searching one merchant across a member's cards.
-- The cursor is (occurred_on, id), matching idx_transactions_feed, because an OFFSET moves
-- under a feed that is being written to.
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
       -- The escape makes % and _ literal: a search for "50%" must not match every row.
       OR lower(t.note) LIKE '%' || replace(replace(replace(lower(@query), '\', '\\'), '%', '\%'), '_', '\_') || '%' ESCAPE '\'
       OR lower(t.merchant) LIKE '%' || replace(replace(replace(lower(@query), '\', '\\'), '%', '\%'), '_', '\_') || '%' ESCAPE '\')
  AND (sqlc.narg('cursor_date')::date IS NULL
       OR t.occurred_on < @cursor_date
       OR (t.occurred_on = @cursor_date AND t.id < sqlc.narg('cursor_id')::uuid))
ORDER BY t.occurred_on DESC, t.id DESC
LIMIT sqlc.arg('page_size')::int;

-- SumVisibleTransactions is the period total behind the feed header and the Home headline. It
-- covers the whole period, not the page the feed happens to be showing.
--
-- With a kind filter every row is on the same side of the ledger and the sum is a magnitude.
-- Without one the two sides are netted — expenses minus income — because adding "spent 5,000"
-- to "earned 20,000" produces a number that describes nothing.
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
       -- The escape makes % and _ literal: a search for "50%" must not match every row.
       OR lower(t.note) LIKE '%' || replace(replace(replace(lower(@query), '\', '\\'), '%', '\%'), '_', '\_') || '%' ESCAPE '\'
       OR lower(t.merchant) LIKE '%' || replace(replace(replace(lower(@query), '\', '\\'), '%', '\%'), '_', '\_') || '%' ESCAPE '\');

-- SumByGroup backs the donut and the Home group rows in one pass. The join to categories is a
-- LEFT one: a transaction with no category still spent money, and dropping it here would make
-- the sum of the group rows smaller than the period total the same period reports.
--
-- It nets income against expense on an unfiltered kind exactly as SumVisibleTransactions does,
-- so GetHomeSummary.period_total and ListTransactions.period_total agree for the same window.
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

-- SumByMember is the split bar on the member screen and the stacked series on the charts
-- screen; group_id is carried so one pass fills both the per-member totals and the per-group
-- member segments.
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

-- SumDailyTotals feeds the bucketed series: one row per calendar day, bucketed in Go so the
-- week/month/year switch does not need three queries.
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

-- SumBudgetSpend is one budget's window, evaluated against either its group or its single
-- category and optionally narrowed to one member. It is deliberately its own query rather
-- than a filter on SumByGroup: a budget's window is not the screen's period.
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
