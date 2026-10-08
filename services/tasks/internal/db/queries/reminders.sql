-- name: UpsertReminder :exec
INSERT INTO task_reminders (family_id, user_id, kind, item_id, occurrence, remind_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (family_id, user_id, kind, item_id, occurrence) DO UPDATE
SET remind_at = EXCLUDED.remind_at
WHERE task_reminders.acked_at IS NULL AND task_reminders.snoozed_until IS NULL;

-- name: DeletePendingRemindersForItem :exec
DELETE FROM task_reminders
WHERE family_id = $1 AND kind = $2 AND item_id = $3 AND acked_at IS NULL;

-- name: DeletePendingRemindersForItemExcept :exec
DELETE FROM task_reminders
WHERE family_id = $1 AND kind = $2 AND item_id = $3 AND acked_at IS NULL
  AND NOT (user_id = ANY(COALESCE(sqlc.arg(keep_users)::uuid[], '{}')) AND occurrence = sqlc.arg(keep_occurrence)::text);

-- name: GetReminder :one
SELECT * FROM task_reminders
WHERE id = $1 AND family_id = $2 AND user_id = $3;

-- name: ListDueRemindersForUser :many
SELECT r.* FROM task_reminders r
WHERE r.family_id = sqlc.arg(family_id) AND r.user_id = sqlc.arg(user_id)
  AND r.acked_at IS NULL AND r.remind_at <= sqlc.arg(now)
  AND NOT EXISTS (SELECT 1 FROM reminder_deliveries d WHERE d.reminder_id = r.id AND d.channel = sqlc.arg(channel)::text)
ORDER BY r.remind_at, r.id
LIMIT 50;

-- name: ListDueReminders :many
SELECT r.* FROM task_reminders r
WHERE r.acked_at IS NULL AND r.remind_at <= sqlc.arg(now)
  AND NOT EXISTS (SELECT 1 FROM reminder_deliveries d WHERE d.reminder_id = r.id AND d.channel = sqlc.arg(channel)::text)
ORDER BY r.remind_at, r.id
LIMIT sqlc.arg(max_rows);

-- name: AckReminder :execrows
UPDATE task_reminders
SET acked_at = NOW()
WHERE id = $1 AND family_id = $2 AND user_id = $3 AND acked_at IS NULL;

-- name: SnoozeReminder :one
UPDATE task_reminders
SET remind_at = $4, snoozed_until = $4, acked_at = NULL
WHERE id = $1 AND family_id = $2 AND user_id = $3
RETURNING *;

-- name: ClearDeliveries :exec
DELETE FROM reminder_deliveries
WHERE reminder_id = $1 AND family_id = $2;

-- name: RecordDelivery :exec
INSERT INTO reminder_deliveries (reminder_id, family_id, channel)
VALUES ($1, $2, $3)
ON CONFLICT (reminder_id, channel) DO UPDATE SET delivered_at = NOW();

-- name: GetDigestSent :one
SELECT EXISTS (
    SELECT 1 FROM digest_sends WHERE family_id = $1 AND user_id = $2 AND sent_on = $3
);

-- name: MarkDigestSent :exec
INSERT INTO digest_sends (family_id, user_id, sent_on)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;
