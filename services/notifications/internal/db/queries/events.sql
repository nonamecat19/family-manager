-- name: ClaimEvent :execrows
INSERT INTO processed_events (event_id, subject, status, claimed_at)
VALUES (@event_id, @subject, 'claimed', NOW())
ON CONFLICT (event_id) DO UPDATE
SET claimed_at = NOW()
WHERE processed_events.status = 'claimed'
  AND processed_events.claimed_at < NOW() - make_interval(secs => @stale_after_seconds::float8);

-- name: EventStatus :one
SELECT status FROM processed_events
WHERE event_id = $1;

-- name: CompleteEvent :exec
UPDATE processed_events
SET status = 'done', processed_at = NOW()
WHERE event_id = $1;

-- name: ReleaseEvent :exec
DELETE FROM processed_events
WHERE event_id = $1 AND status = 'claimed';

-- name: PruneProcessedEvents :execrows
DELETE FROM processed_events
WHERE processed_at < $1;
