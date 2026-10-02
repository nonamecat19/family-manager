-- name: IsEventProcessed :one
SELECT EXISTS (SELECT 1 FROM processed_events WHERE event_id = $1);

-- name: MarkEventProcessed :exec
INSERT INTO processed_events (event_id, subject)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: PruneProcessedEvents :execrows
DELETE FROM processed_events
WHERE processed_at < $1;
