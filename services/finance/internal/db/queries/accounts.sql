-- name: CreateAccount :one
INSERT INTO accounts (
    family_id, name, type, currency_code, opening_balance_minor, color, icon, sort_order
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- A balance is never stored: it is the opening balance plus everything that moved. Storing it
-- would mean two sources of truth, and the stored one is always the stale one.
--
-- name: ListAccountsWithBalance :many
SELECT
    a.*,
    (
        a.opening_balance_minor
        + COALESCE((
            SELECT SUM(
                CASE t.type
                    WHEN 'income' THEN t.amount_minor
                    ELSE -t.amount_minor          -- expense and outgoing transfer both leave
                END
            )
            FROM transactions t
            WHERE t.account_id = a.id
        ), 0)
        + COALESCE((
            SELECT SUM(t.amount_minor)            -- incoming side of a transfer
            FROM transactions t
            WHERE t.counter_account_id = a.id
        ), 0)
    )::bigint AS balance_minor
FROM accounts a
WHERE a.family_id = @family_id
  AND (@include_archived::bool OR a.archived = FALSE)
ORDER BY a.sort_order, a.created_at;

-- name: GetAccountWithBalance :one
SELECT
    a.*,
    (
        a.opening_balance_minor
        + COALESCE((
            SELECT SUM(CASE t.type WHEN 'income' THEN t.amount_minor ELSE -t.amount_minor END)
            FROM transactions t
            WHERE t.account_id = a.id
        ), 0)
        + COALESCE((
            SELECT SUM(t.amount_minor)
            FROM transactions t
            WHERE t.counter_account_id = a.id
        ), 0)
    )::bigint AS balance_minor
FROM accounts a
WHERE a.id = @id AND a.family_id = @family_id;

-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = @id AND family_id = @family_id;

-- name: UpdateAccount :one
UPDATE accounts
SET name       = @name,
    type       = @type,
    color      = @color,
    icon       = @icon,
    archived   = @archived,
    sort_order = @sort_order,
    updated_at = NOW()
WHERE id = @id AND family_id = @family_id
RETURNING *;

-- name: DeleteAccount :execrows
DELETE FROM accounts
WHERE id = @id AND family_id = @family_id;

-- name: CountAccountTransactions :one
SELECT count(*) FROM transactions
WHERE account_id = @account_id OR counter_account_id = @account_id;
