-- name: CreateTransaction :one
INSERT INTO transactions (
    family_id, account_id, counter_account_id, category_id, type,
    amount_minor, currency_code, note, occurred_on, created_by_user_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions
WHERE id = @id AND family_id = @family_id;

-- Keyset pagination on (occurred_on, id): OFFSET drifts when a row is inserted mid-scroll,
-- and the ledger is written to while it is being read.
--
-- name: ListTransactions :many
SELECT * FROM transactions
WHERE family_id = @family_id
  AND occurred_on >= @from_date
  AND occurred_on <= @to_date
  AND (
      cardinality(@account_ids::uuid[]) = 0
      OR account_id = ANY (@account_ids::uuid[])
      OR counter_account_id = ANY (@account_ids::uuid[])
  )
  AND (cardinality(@category_ids::uuid[]) = 0 OR category_id = ANY (@category_ids::uuid[]))
  AND (@type::text = '' OR type = @type::text)
  AND (@search::text = '' OR note ILIKE '%' || @search::text || '%')
  AND (
      NOT @use_cursor::bool
      OR (occurred_on, id) < (@cursor_date::date, @cursor_id::uuid)
  )
ORDER BY occurred_on DESC, id DESC
LIMIT @page_size;

-- name: UpdateTransaction :one
UPDATE transactions
SET account_id         = @account_id,
    counter_account_id = @counter_account_id,
    category_id        = @category_id,
    type               = @type,
    amount_minor       = @amount_minor,
    currency_code      = @currency_code,
    note               = @note,
    occurred_on        = @occurred_on,
    updated_at         = NOW()
WHERE id = @id AND family_id = @family_id
RETURNING *;

-- name: DeleteTransaction :execrows
DELETE FROM transactions
WHERE id = @id AND family_id = @family_id;

-- Transfers are excluded from both totals: moving money between your own accounts is not
-- income and not spending.
--
-- name: GetSummary :one
SELECT
    COALESCE(SUM(amount_minor) FILTER (WHERE type = 'income'), 0)::bigint  AS income_minor,
    COALESCE(SUM(amount_minor) FILTER (WHERE type = 'expense'), 0)::bigint AS expense_minor
FROM transactions
WHERE family_id = @family_id
  AND occurred_on >= @from_date
  AND occurred_on <= @to_date
  AND (
      cardinality(@account_ids::uuid[]) = 0
      OR account_id = ANY (@account_ids::uuid[])
  );

-- name: GetCategoryBreakdown :many
SELECT
    c.id                                  AS category_id,
    c.name                                AS category_name,
    c.color                               AS color,
    COALESCE(SUM(t.amount_minor), 0)::bigint AS total_minor,
    COUNT(t.id)::bigint                   AS transaction_count
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.family_id = @family_id
  AND t.occurred_on >= @from_date
  AND t.occurred_on <= @to_date
  AND t.type = @type::text
  AND (
      cardinality(@account_ids::uuid[]) = 0
      OR t.account_id = ANY (@account_ids::uuid[])
  )
GROUP BY c.id, c.name, c.color
ORDER BY total_minor DESC;
