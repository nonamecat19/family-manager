CREATE TABLE IF NOT EXISTS processed_events (
    event_id   TEXT PRIMARY KEY,
    subject    TEXT NOT NULL,
    status     TEXT NOT NULL CHECK (status IN ('claimed','done')),
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS processed_events_claimed_at ON processed_events (claimed_at) WHERE status = 'claimed';