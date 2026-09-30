CREATE TABLE IF NOT EXISTS captured_notifications (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id        UUID NOT NULL,
    user_id          UUID NOT NULL,
    device_id        TEXT NOT NULL CHECK (length(device_id) BETWEEN 1 AND 256),
    notification_key TEXT NOT NULL CHECK (length(notification_key) BETWEEN 1 AND 512),
    package_name     TEXT NOT NULL CHECK (length(package_name) BETWEEN 1 AND 255),
    title_sealed     BYTEA,
    text_sealed      BYTEA,
    posted_at        TIMESTAMPTZ NOT NULL,
    received_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (family_id, user_id, device_id, notification_key),
    UNIQUE (id, family_id, user_id)
);

CREATE INDEX IF NOT EXISTS captured_notifications_member_received ON captured_notifications (family_id, user_id, received_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS captured_notifications_unpurged_received ON captured_notifications (received_at) WHERE title_sealed IS NOT NULL OR text_sealed IS NOT NULL;

CREATE TABLE IF NOT EXISTS suggestions (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id              UUID NOT NULL,
    user_id                UUID NOT NULL,
    notification_id        UUID NOT NULL UNIQUE,
    status                 TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','confirmed','dismissed')),
    amount_minor           BIGINT NOT NULL CHECK (amount_minor >= 0),
    currency_code          TEXT NOT NULL CHECK (currency_code ~ '^[A-Z]{3}$'),
    direction              TEXT NOT NULL CHECK (direction IN ('debit','credit')),
    merchant               TEXT NOT NULL DEFAULT '' CHECK (length(merchant) <= 512),
    card_hint              TEXT NOT NULL DEFAULT '' CHECK (length(card_hint) <= 32),
    balance_minor          BIGINT,
    balance_currency_code  TEXT CHECK (balance_currency_code IS NULL OR balance_currency_code ~ '^[A-Z]{3}$'),
    parser_id              TEXT NOT NULL CHECK (length(btrim(parser_id)) BETWEEN 1 AND 128),
    occurred_at            TIMESTAMPTZ NOT NULL,
    suggested_account_id   UUID,
    suggested_category_id  UUID,
    finance_transaction_id UUID,
    decided_at             TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT suggestions_decided_iff_not_pending CHECK ((status = 'pending') = (decided_at IS NULL)),
    CONSTRAINT suggestions_transaction_iff_confirmed CHECK ((status = 'confirmed') = (finance_transaction_id IS NOT NULL)),
    CONSTRAINT suggestions_balance_complete CHECK ((balance_minor IS NULL) = (balance_currency_code IS NULL)),
    FOREIGN KEY (notification_id, family_id, user_id) REFERENCES captured_notifications (id, family_id, user_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS suggestions_member_status_occurred ON suggestions (family_id, user_id, status, occurred_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS suggestions_member_occurred ON suggestions (family_id, user_id, occurred_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS suggestions_decided ON suggestions (decided_at) WHERE status <> 'pending';
CREATE INDEX IF NOT EXISTS suggestions_pending_created ON suggestions (created_at) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS capture_settings (
    family_id        UUID NOT NULL,
    user_id          UUID NOT NULL,
    enabled          BOOLEAN NOT NULL DEFAULT false,
    allowed_packages TEXT[] NOT NULL DEFAULT '{}' CHECK (cardinality(allowed_packages) <= 64),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (family_id, user_id)
);

CREATE TABLE IF NOT EXISTS prefill_rules (
    family_id   UUID NOT NULL,
    user_id     UUID NOT NULL,
    match_kind  TEXT NOT NULL CHECK (match_kind IN ('card_hint','merchant')),
    match_value TEXT NOT NULL CHECK (length(match_value) BETWEEN 1 AND 512),
    account_id  UUID,
    category_id UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (family_id, user_id, match_kind, match_value)
);

CREATE TABLE IF NOT EXISTS deleted_notification_keys (
    family_id        UUID NOT NULL,
    user_id          UUID NOT NULL,
    device_id        TEXT NOT NULL,
    notification_key TEXT NOT NULL,
    deleted_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (family_id, user_id, device_id, notification_key)
);
