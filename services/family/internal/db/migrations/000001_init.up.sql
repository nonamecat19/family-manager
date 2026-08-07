-- services/family owns the household graph. user_id columns reference users in
-- services/auth and therefore carry NO foreign key: crossing a service boundary in SQL is
-- what the contract in libs/proto exists to prevent.

CREATE TABLE IF NOT EXISTS families (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    owner_user_id UUID NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_families_owner ON families (owner_user_id);

CREATE TABLE IF NOT EXISTS family_members (
    family_id    UUID NOT NULL REFERENCES families (id) ON DELETE CASCADE,
    user_id      UUID NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    email        TEXT NOT NULL DEFAULT '',
    role         TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (family_id, user_id)
);

-- One family per user, for now: the apps have no family switcher, and a partial unique index
-- is cheaper to drop later than a bad row is to untangle.
CREATE UNIQUE INDEX IF NOT EXISTS idx_family_members_one_family_per_user
    ON family_members (user_id);

CREATE TABLE IF NOT EXISTS family_invitations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES families (id) ON DELETE CASCADE,
    inviter_user_id UUID NOT NULL,
    email           TEXT NOT NULL,
    role            TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    -- Only the hash is stored; the plaintext token is shown to the inviter once and never
    -- persisted, so a database leak cannot be replayed into a family.
    token_hash      TEXT NOT NULL UNIQUE,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'accepted', 'revoked', 'expired')),
    expires_at      TIMESTAMPTZ NOT NULL,
    accepted_by     UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_family_invitations_family ON family_invitations (family_id);
CREATE INDEX IF NOT EXISTS idx_family_invitations_status ON family_invitations (family_id, status);
