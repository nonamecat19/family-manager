-- name: UpsertPushToken :execrows
INSERT INTO push_tokens (token, user_id, family_id, platform, app, device_id)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (token) DO UPDATE
SET user_id    = EXCLUDED.user_id,
    family_id  = EXCLUDED.family_id,
    platform   = EXCLUDED.platform,
    app        = EXCLUDED.app,
    device_id  = EXCLUDED.device_id,
    updated_at = NOW()
WHERE push_tokens.user_id = EXCLUDED.user_id
   OR (push_tokens.device_id <> '' AND push_tokens.device_id = EXCLUDED.device_id);

-- name: DeleteUserPushToken :execrows
DELETE FROM push_tokens
WHERE token = $1 AND user_id = $2;

-- name: DeleteDeadPushTokens :execrows
DELETE FROM push_tokens p
USING (
    SELECT unnest(@tokens::text[]) AS token,
           unnest(@sent_at::timestamptz[]) AS sent_at
) d
WHERE p.token = d.token
  AND p.updated_at < d.sent_at;

-- name: ListFamilyPushTokens :many
SELECT * FROM push_tokens
WHERE family_id = $1
ORDER BY user_id, created_at;

-- name: ListUserPushTokens :many
SELECT * FROM push_tokens
WHERE user_id = $1
ORDER BY created_at;

-- name: SetUserFamily :exec
UPDATE push_tokens
SET family_id = $2, updated_at = NOW()
WHERE user_id = $1;

-- name: ClearUserFamily :exec
UPDATE push_tokens
SET family_id = NULL, updated_at = NOW()
WHERE user_id = $1 AND family_id = $2;
