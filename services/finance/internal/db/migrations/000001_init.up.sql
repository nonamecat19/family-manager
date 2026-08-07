-- services/finance owns the ledger. family_id and user_id reference rows owned by
-- services/family and services/auth and therefore carry NO foreign key — a cross-service
-- join is exactly what libs/proto exists to replace.
--
-- Money is stored as (amount_minor BIGINT, currency_code TEXT). No NUMERIC, no float: minor
-- units are exact, and a currency without its code is not an amount.

CREATE TABLE IF NOT EXISTS accounts (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL,
    name                  TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    type                  TEXT NOT NULL DEFAULT 'cash'
                          CHECK (type IN ('cash', 'card', 'bank', 'savings', 'debt')),
    currency_code         TEXT NOT NULL CHECK (currency_code ~ '^[A-Z]{3}$'),
    opening_balance_minor BIGINT NOT NULL DEFAULT 0,
    color                 TEXT NOT NULL DEFAULT '',
    icon                  TEXT NOT NULL DEFAULT '',
    archived              BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order            INTEGER NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_accounts_family ON accounts (family_id, archived, sort_order);

CREATE TABLE IF NOT EXISTS categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id  UUID NOT NULL,
    name       TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    -- kind mirrors finance.v1.TransactionType; 'transfer' is not a category kind, transfers
    -- are categorised by the accounts they join.
    kind       TEXT NOT NULL CHECK (kind IN ('expense', 'income')),
    color      TEXT NOT NULL DEFAULT '',
    icon       TEXT NOT NULL DEFAULT '',
    parent_id  UUID REFERENCES categories (id) ON DELETE SET NULL,
    archived   BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_categories_family ON categories (family_id, kind, archived);
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_family_name_kind
    ON categories (family_id, lower(btrim(name)), kind);

CREATE TABLE IF NOT EXISTS transactions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id          UUID NOT NULL,
    account_id         UUID NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    -- Set only for transfers: the destination account.
    counter_account_id UUID REFERENCES accounts (id) ON DELETE RESTRICT,
    category_id        UUID REFERENCES categories (id) ON DELETE RESTRICT,
    type               TEXT NOT NULL CHECK (type IN ('expense', 'income', 'transfer')),
    -- Always positive; `type` carries the direction. A negative expense is a data bug, not a
    -- refund — refunds are income.
    amount_minor       BIGINT NOT NULL CHECK (amount_minor > 0),
    currency_code      TEXT NOT NULL CHECK (currency_code ~ '^[A-Z]{3}$'),
    note               TEXT NOT NULL DEFAULT '',
    occurred_on        DATE NOT NULL,
    created_by_user_id UUID NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- A transfer needs a destination and no category; income/expense need a category and no
    -- destination. Enforced here so no handler bug can write a shape reports cannot read.
    CONSTRAINT transactions_shape CHECK (
        (type = 'transfer' AND counter_account_id IS NOT NULL AND counter_account_id <> account_id)
        OR (type <> 'transfer' AND counter_account_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_transactions_family_date
    ON transactions (family_id, occurred_on DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_account ON transactions (account_id, occurred_on);
CREATE INDEX IF NOT EXISTS idx_transactions_category ON transactions (category_id, occurred_on);
