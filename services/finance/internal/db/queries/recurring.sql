-- A recurring payment is attached to an account, so it inherits that account's visibility:
-- the schedule's name, amount and cadence say as much about a private account as a
-- transaction does. Every read therefore joins accounts and carries the same predicate the
-- rest of this directory carries
--
--     (a.visibility = 'shared' OR a.owner_member_id = @viewer_member_id)
--
-- and is named Visible* so a handler reaching for an unscoped read has to notice there isn't
-- one. Writes are gated by reading the row through GetVisibleRecurringPayment first — the
-- same "the read is the guard" rule accounts.sql states.

-- name: ListVisibleRecurringPayments :many
SELECT r.*
FROM recurring_payments r
JOIN accounts a ON a.id = r.account_id
WHERE r.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND (sqlc.arg('include_inactive')::bool OR r.active)
ORDER BY r.next_due_on, r.created_at;

-- GetVisibleRecurringPayment answers NotFound for a schedule on another member's private
-- account, the same answer as an id that never existed: "this id exists but is not yours" is
-- itself a leak, and a write RPC that skipped this read would be a read of exactly what the
-- boundary hides.
-- name: GetVisibleRecurringPayment :one
SELECT r.*
FROM recurring_payments r
JOIN accounts a ON a.id = r.account_id
WHERE r.id = $1
  AND r.family_id = $2
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid);

-- name: CreateRecurringPayment :one
INSERT INTO recurring_payments (family_id, name, amount_minor, currency_code, type,
    category_id, account_id, member_id, interval_count, interval_unit, day_of_month,
    day_of_week, next_due_on, end_on, auto_post)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: UpdateRecurringPayment :one
UPDATE recurring_payments
SET name           = COALESCE(sqlc.narg('name')::text, name),
    amount_minor   = COALESCE(sqlc.narg('amount_minor')::bigint, amount_minor),
    -- Same rule as quick_templates: the schedule is denominated in its account's currency, so
    -- moving it to another account moves the currency with it.
    currency_code  = COALESCE(sqlc.narg('currency_code')::text, currency_code),
    category_id    = COALESCE(sqlc.narg('category_id')::uuid, category_id),
    account_id     = COALESCE(sqlc.narg('account_id')::uuid, account_id),
    member_id      = COALESCE(sqlc.narg('member_id')::uuid, member_id),
    interval_count = COALESCE(sqlc.narg('interval_count')::int, interval_count),
    interval_unit  = COALESCE(sqlc.narg('interval_unit')::text, interval_unit),
    day_of_month   = COALESCE(sqlc.narg('day_of_month')::int, day_of_month),
    day_of_week    = COALESCE(sqlc.narg('day_of_week')::text, day_of_week),
    next_due_on    = COALESCE(sqlc.narg('next_due_on')::date, next_due_on),
    end_on         = COALESCE(sqlc.narg('end_on')::date, end_on),
    auto_post      = COALESCE(sqlc.narg('auto_post')::bool, auto_post),
    active         = COALESCE(sqlc.narg('active')::bool, active),
    updated_at     = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: AdvanceRecurringPayment :one
UPDATE recurring_payments
SET next_due_on = $3, last_posted_on = sqlc.narg('last_posted_on')::date, updated_at = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: DeleteRecurringPayment :execrows
DELETE FROM recurring_payments
WHERE id = $1 AND family_id = $2;
