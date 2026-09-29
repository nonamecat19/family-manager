CREATE TABLE IF NOT EXISTS chat_states (
    bot              TEXT   NOT NULL,
    telegram_user_id BIGINT NOT NULL,
    kind             TEXT   NOT NULL,
    payload          JSONB  NOT NULL DEFAULT '{}'::jsonb,
    expires_at       TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot, telegram_user_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_states_expiry ON chat_states (expires_at);
