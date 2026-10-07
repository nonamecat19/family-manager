-- name: GetFinanceSettings :one
SELECT * FROM finance_settings
WHERE family_id = $1;

-- name: BootstrapFinanceSettings :one
INSERT INTO finance_settings (family_id, base_currency_code, timezone, week_starts_on)
VALUES ($1, $2, $3, $4)
ON CONFLICT (family_id) DO UPDATE SET family_id = EXCLUDED.family_id
RETURNING *;

-- name: UpdateFinanceSettings :one
UPDATE finance_settings
SET base_currency_code = COALESCE(sqlc.narg('base_currency_code')::text, base_currency_code),
    timezone           = COALESCE(sqlc.narg('timezone')::text, timezone),
    week_starts_on     = COALESCE(sqlc.narg('week_starts_on')::text, week_starts_on),
    overspend_notifications_enabled =
        COALESCE(sqlc.narg('overspend_notifications_enabled')::bool, overspend_notifications_enabled),
    pin_lock_enabled   = COALESCE(sqlc.narg('pin_lock_enabled')::bool, pin_lock_enabled),
    updated_at         = NOW()
WHERE family_id = $1
RETURNING *;

-- name: SetOverspendNotifications :one
UPDATE finance_settings
SET overspend_notifications_enabled = $2, updated_at = NOW()
WHERE family_id = $1
RETURNING *;
