DELETE FROM processed_events
WHERE status = 'claimed';

ALTER TABLE processed_events
    DROP COLUMN IF EXISTS claimed_at,
    DROP COLUMN IF EXISTS status;
