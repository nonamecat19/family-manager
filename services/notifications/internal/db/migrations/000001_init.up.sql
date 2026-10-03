CREATE TABLE IF NOT EXISTS push_tokens (
    token      TEXT PRIMARY KEY,
    user_id    UUID NOT NULL,
    family_id  UUID,
    platform   TEXT NOT NULL CHECK (platform IN ('ios', 'android')),
    app        TEXT NOT NULL CHECK (app IN ('notes', 'finance', 'recipes')),
    device_id  TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_push_tokens_user ON push_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_push_tokens_family ON push_tokens (family_id)
    WHERE family_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS notification_mutes (
    user_id    UUID NOT NULL,
    topic      TEXT NOT NULL CHECK (length(topic) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, topic)
);

CREATE TABLE IF NOT EXISTS processed_events (
    event_id     TEXT PRIMARY KEY,
    subject      TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS push_tickets (
    id         TEXT PRIMARY KEY,
    token      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_push_tickets_created ON push_tickets (created_at);
