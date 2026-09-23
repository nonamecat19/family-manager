ALTER TABLE processed_events
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'done'
        CHECK (status IN ('claimed', 'done')),
    ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
