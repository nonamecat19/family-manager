-- name: ListReminders :many
SELECT * FROM reminders
WHERE family_id = $1 AND user_id = $2
  AND (sqlc.arg('include_disabled')::bool OR enabled)
ORDER BY due_at NULLS LAST, created_at;

-- name: GetReminder :one
SELECT * FROM reminders
WHERE id = $1 AND family_id = $2 AND user_id = $3;

-- name: CreateReminder :one
INSERT INTO reminders (family_id, user_id, kind, title, due_at, repeat_interval, repeat_unit, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateReminder :one
UPDATE reminders
SET kind = $4, title = $5, due_at = $6, repeat_interval = $7, repeat_unit = $8,
    enabled = $9, updated_at = NOW()
WHERE id = $1 AND family_id = $2 AND user_id = $3
RETURNING *;

-- name: DeleteReminder :execrows
DELETE FROM reminders
WHERE id = $1 AND family_id = $2 AND user_id = $3;
