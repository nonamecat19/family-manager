-- name: ListVisibleSubscriptions :many
SELECT sqlc.embed(s), c.group_id AS group_id,
    COALESCE(p.paid_minor, 0)::bigint AS paid_minor,
    COALESCE(p.payments, 0)::bigint AS payments
FROM subscriptions s
JOIN accounts a ON a.id = s.account_id
JOIN categories c ON c.id = s.category_id
LEFT JOIN LATERAL (
    SELECT SUM(t.amount_minor) AS paid_minor, COUNT(*) AS payments
    FROM transactions t
    JOIN accounts ta ON ta.id = t.account_id
    WHERE t.family_id = s.family_id
      AND t.category_id = s.category_id
      AND t.type = 'expense'
      AND t.currency_code = s.currency_code
      AND (ta.visibility = 'shared' OR ta.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
) p ON TRUE
WHERE s.family_id = $1
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid)
  AND (sqlc.arg('include_inactive')::bool OR s.status = 'active')
ORDER BY s.status = 'active' DESC, s.next_due_on, s.created_at;

-- name: GetVisibleSubscription :one
SELECT s.*
FROM subscriptions s
JOIN accounts a ON a.id = s.account_id
WHERE s.id = $1
  AND s.family_id = $2
  AND (a.visibility = 'shared' OR a.owner_member_id = sqlc.arg('viewer_member_id')::uuid);

-- name: CreateSubscription :one
INSERT INTO subscriptions (family_id, name, amount_minor, currency_code, type,
    category_id, account_id, member_id, created_by_user_id,
    interval_count, interval_unit, day_of_month, day_of_week,
    next_due_on, end_on, auto_post)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: UpdateSubscription :one
UPDATE subscriptions
SET name            = COALESCE(sqlc.narg('name')::text, name),
    amount_minor    = COALESCE(sqlc.narg('amount_minor')::bigint, amount_minor),
    category_id     = COALESCE(sqlc.narg('category_id')::uuid, category_id),
    account_id      = COALESCE(sqlc.narg('account_id')::uuid, account_id),
    member_id       = COALESCE(sqlc.narg('member_id')::uuid, member_id),
    interval_count  = COALESCE(sqlc.narg('interval_count')::int, interval_count),
    interval_unit   = COALESCE(sqlc.narg('interval_unit')::text, interval_unit),
    day_of_month    = COALESCE(sqlc.narg('day_of_month')::int, day_of_month),
    day_of_week     = COALESCE(sqlc.narg('day_of_week')::text, day_of_week),
    next_due_on     = COALESCE(sqlc.narg('next_due_on')::date, next_due_on),
    end_on          = COALESCE(sqlc.narg('end_on')::date, end_on),
    auto_post       = COALESCE(sqlc.narg('auto_post')::bool, auto_post),
    active          = COALESCE(sqlc.narg('active')::bool, active),
    status          = COALESCE(sqlc.narg('status')::text, status),
    updated_at      = NOW()
WHERE id = $1 AND family_id = $2
RETURNING *;

-- name: DeleteSubscription :execrows
DELETE FROM subscriptions
WHERE id = $1 AND family_id = $2;

-- name: ListDueSubscriptions :many
SELECT id, family_id, category_id, next_due_on, amount_minor, currency_code, type,
       account_id, member_id, created_by_user_id, interval_count, interval_unit,
       day_of_month, day_of_week, name
FROM subscriptions
WHERE status = 'active'
  AND active = true
  AND auto_post = true
  AND next_due_on <= sqlc.arg('latest_due_on')::date
  AND (end_on IS NULL OR end_on >= next_due_on)
ORDER BY next_due_on
LIMIT sqlc.arg('max_rows')::int;

-- name: LockDueSubscription :one
SELECT * FROM subscriptions
WHERE id = $1 AND status = 'active' AND active = true
FOR UPDATE SKIP LOCKED;

-- name: SumSubscriptionPayments :one
SELECT COALESCE(SUM(amount_minor), 0)::bigint AS paid_minor, COUNT(*)::bigint AS payments
FROM transactions
WHERE family_id = $1 AND category_id = $2 AND type = 'expense' AND currency_code = $3;

-- name: AdvanceSubscription :one
UPDATE subscriptions
SET next_due_on = sqlc.arg('next_due_on')::date,
    last_posted_on = sqlc.arg('last_posted_on')::date,
    updated_at  = NOW()
WHERE id = $1 AND next_due_on = sqlc.arg('expected_due_on')::date AND status = 'active' AND active = true
RETURNING *;

-- name: GetSubscriptionByID :one
SELECT * FROM subscriptions
WHERE id = $1;

-- name: ClaimSubscriptionOccurrence :execrows
INSERT INTO subscription_occurrences (subscription_id, due_on)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
