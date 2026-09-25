CREATE TABLE IF NOT EXISTS telegram_links (
    bot                TEXT   NOT NULL,
    telegram_user_id   BIGINT NOT NULL,
    user_id            UUID   NOT NULL,
    telegram_username  TEXT   NOT NULL DEFAULT '',
    chat_id            BIGINT NOT NULL,
    access_token       BYTEA  NOT NULL,
    access_expires_at  TIMESTAMPTZ NOT NULL,
    refresh_token      BYTEA  NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot, telegram_user_id)
);

CREATE INDEX IF NOT EXISTS idx_telegram_links_user ON telegram_links (user_id);

CREATE TABLE IF NOT EXISTS bot_offsets (
    bot        TEXT PRIMARY KEY,
    offset_id  BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
