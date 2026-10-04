CREATE TABLE IF NOT EXISTS known_members (
    family_id    UUID NOT NULL,
    user_id      UUID NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    email        TEXT NOT NULL DEFAULT '',
    seen_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (family_id, user_id)
);
