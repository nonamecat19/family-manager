CREATE TABLE IF NOT EXISTS chains (
    id         UUID PRIMARY KEY,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO chains (id, revoked_at)
SELECT chain_id, MAX(revoked_at)
FROM refresh_tokens
GROUP BY chain_id
ON CONFLICT (id) DO NOTHING;
