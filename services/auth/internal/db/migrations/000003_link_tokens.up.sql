CREATE TABLE IF NOT EXISTS link_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider   TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    external_id TEXT,
    used_at    TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_link_tokens_user ON link_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_link_tokens_expiry ON link_tokens (expires_at);
