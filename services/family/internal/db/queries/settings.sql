-- name: GetUserSettings :one
SELECT * FROM user_settings
WHERE user_id = $1;

-- name: UpsertUserSettings :one
INSERT INTO user_settings (user_id, locale, timezone)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE
SET locale     = COALESCE(NULLIF(EXCLUDED.locale, ''), user_settings.locale),
    timezone   = COALESCE(NULLIF(EXCLUDED.timezone, ''), user_settings.timezone),
    updated_at = NOW()
RETURNING *;
