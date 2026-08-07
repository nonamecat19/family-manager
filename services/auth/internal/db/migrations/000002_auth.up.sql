-- The column has always held an argon2id hash, never a password. Naming it `password` invites
-- someone to write a plaintext into it; renaming is cheap now and expensive after the first
-- deployment.
ALTER TABLE users RENAME COLUMN password TO password_hash;

-- Case-insensitive uniqueness. Without it "Ada@example.com" and "ada@example.com" are two
-- accounts, and login picks whichever index scan happens to return first.
--
-- The UNIQUE in 000001 created a constraint, and the index behind a constraint cannot be
-- dropped directly — DROP INDEX fails with 2BP01. Dropping the constraint takes the index
-- with it. The DROP INDEX afterwards covers a database where the same name exists as a bare
-- index instead.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
DROP INDEX IF EXISTS users_email_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (lower(email));

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Only the SHA-256 of the token is stored. A refresh token is a bearer credential with a
    -- long life; a database leak must not be replayable into live sessions.
    token_hash TEXT NOT NULL UNIQUE,
    -- Rotation chain: each refresh mints a successor and marks its parent used. A second use
    -- of an already-used token means the token was stolen, and the whole chain is revoked.
    chain_id   UUID NOT NULL,
    used_at    TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_chain ON refresh_tokens (chain_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expiry ON refresh_tokens (expires_at);
