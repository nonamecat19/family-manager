CREATE TABLE IF NOT EXISTS user_settings (
    user_id    UUID PRIMARY KEY,
    locale     TEXT NOT NULL DEFAULT 'uk',
    timezone   TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
