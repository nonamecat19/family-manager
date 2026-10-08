-- name: GetGoogleConnection :one
SELECT * FROM google_connections
WHERE family_id = $1 AND user_id = $2;

-- name: UpsertGoogleConnection :one
INSERT INTO google_connections (family_id, user_id, email, refresh_token)
VALUES ($1, $2, $3, $4)
ON CONFLICT (family_id, user_id) DO UPDATE
SET email = EXCLUDED.email, refresh_token = EXCLUDED.refresh_token,
    calendar_id = CASE WHEN google_connections.email = EXCLUDED.email THEN google_connections.calendar_id ELSE '' END,
    calendar_name = CASE WHEN google_connections.email = EXCLUDED.email THEN google_connections.calendar_name ELSE '' END,
    sync_token = CASE WHEN google_connections.email = EXCLUDED.email THEN google_connections.sync_token ELSE '' END,
    last_error = '', updated_at = NOW()
RETURNING *;

-- name: SetGoogleCalendar :one
UPDATE google_connections
SET calendar_id = $3, calendar_name = $4, sync_token = '', updated_at = NOW()
WHERE family_id = $1 AND user_id = $2
RETURNING *;

-- name: SetGoogleSyncState :exec
UPDATE google_connections
SET sync_token = $3, last_synced_at = $4, last_error = $5, updated_at = NOW()
WHERE family_id = $1 AND user_id = $2;

-- name: SetGoogleError :exec
UPDATE google_connections
SET last_error = $3, updated_at = NOW()
WHERE family_id = $1 AND user_id = $2;

-- name: DeleteGoogleConnection :execrows
DELETE FROM google_connections
WHERE family_id = $1 AND user_id = $2;

-- name: ListFamilyGoogleConnections :many
SELECT * FROM google_connections
WHERE family_id = $1 AND calendar_id <> ''
ORDER BY user_id;

-- name: ListSyncableGoogleConnections :many
SELECT * FROM google_connections
WHERE calendar_id <> ''
ORDER BY last_synced_at NULLS FIRST, family_id, user_id;

-- name: GetCalendarLink :one
SELECT * FROM calendar_links
WHERE family_id = $1 AND user_id = $2 AND kind = $3 AND item_id = $4;

-- name: GetCalendarLinkByEvent :one
SELECT * FROM calendar_links
WHERE family_id = $1 AND user_id = $2 AND calendar_id = $3 AND event_id = $4;

-- name: UpsertCalendarLink :exec
INSERT INTO calendar_links (family_id, user_id, kind, item_id, calendar_id, event_id, etag)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (family_id, user_id, kind, item_id) DO UPDATE
SET calendar_id = EXCLUDED.calendar_id, event_id = EXCLUDED.event_id,
    etag = EXCLUDED.etag, synced_at = NOW();

-- name: DeleteCalendarLink :exec
DELETE FROM calendar_links
WHERE family_id = $1 AND user_id = $2 AND kind = $3 AND item_id = $4;

-- name: ListCalendarLinksForItem :many
SELECT * FROM calendar_links
WHERE family_id = $1 AND kind = $2 AND item_id = $3
ORDER BY user_id;

-- name: CalendarLinkedInFamily :one
SELECT EXISTS (
    SELECT 1 FROM calendar_links
    WHERE family_id = $1 AND kind = $2 AND item_id = $3 AND calendar_id = $4 AND user_id <> $5
);

-- name: DeleteCalendarLinksForUser :exec
DELETE FROM calendar_links
WHERE family_id = $1 AND user_id = $2;

-- name: ListOrphanCalendarLinks :many
SELECT l.* FROM calendar_links l
WHERE l.family_id = $1 AND l.user_id = $2
  AND ((l.kind = 'task' AND NOT EXISTS (SELECT 1 FROM tasks t WHERE t.id = l.item_id AND t.family_id = l.family_id))
    OR (l.kind = 'birthday' AND NOT EXISTS (SELECT 1 FROM birthdays b WHERE b.id = l.item_id AND b.family_id = l.family_id)));
