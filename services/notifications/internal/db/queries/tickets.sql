-- name: CurrentTime :one
SELECT NOW()::timestamptz;

-- name: InsertPushTicket :exec
INSERT INTO push_tickets (id, token)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListDuePushTickets :many
SELECT * FROM push_tickets
WHERE created_at < $1
ORDER BY created_at
LIMIT $2;

-- name: DeletePushTickets :exec
DELETE FROM push_tickets
WHERE id = ANY(@ids::text[]);
