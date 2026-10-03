CREATE TABLE IF NOT EXISTS login_grants (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind             TEXT NOT NULL,
    device_code_hash TEXT NOT NULL UNIQUE,
    user_code_hash   TEXT NOT NULL UNIQUE,
    user_id          UUID REFERENCES users (id) ON DELETE CASCADE,
    approved_at      TIMESTAMPTZ,
    denied_at        TIMESTAMPTZ,
    consumed_at      TIMESTAMPTZ,
    last_polled_at   TIMESTAMPTZ,
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_login_grants_expiry ON login_grants (expires_at);
