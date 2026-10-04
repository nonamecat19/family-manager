CREATE TABLE IF NOT EXISTS login_grants (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind              TEXT NOT NULL,
    device_code_hash  TEXT NOT NULL UNIQUE,
    user_code_hash    TEXT NOT NULL UNIQUE,
    user_id           UUID REFERENCES users (id) ON DELETE CASCADE,
    approver_chain_id UUID,
    root_chain_id     UUID,
    chain_id          UUID,
    approved_at       TIMESTAMPTZ,
    denied_at         TIMESTAMPTZ,
    consumed_at       TIMESTAMPTZ,
    last_polled_at    TIMESTAMPTZ,
    expires_at        TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_login_grants_expiry ON login_grants (expires_at);
CREATE INDEX IF NOT EXISTS idx_login_grants_root_chain ON login_grants (root_chain_id)
    WHERE root_chain_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_login_grants_chain ON login_grants (chain_id)
    WHERE chain_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_login_grants_pending ON login_grants (expires_at)
    WHERE consumed_at IS NULL AND denied_at IS NULL;
