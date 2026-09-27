ALTER TABLE category_groups
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT ''
        CHECK (role IN ('','investments','installments'));

CREATE UNIQUE INDEX IF NOT EXISTS idx_category_groups_family_role
    ON category_groups (family_id, role) WHERE role <> '';

CREATE TABLE IF NOT EXISTS investments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id           UUID NOT NULL,
    name                TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    kind                TEXT NOT NULL DEFAULT 'other'
        CHECK (kind IN ('deposit','stocks','bonds','crypto','real_estate','other')),
    currency_code       TEXT NOT NULL CHECK (length(btrim(currency_code)) = 3),
    category_id         UUID NOT NULL UNIQUE REFERENCES categories (id) ON DELETE RESTRICT,
    current_value_minor BIGINT NOT NULL DEFAULT 0 CHECK (current_value_minor >= 0),
    value_updated_on    DATE,
    archived            BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order          INT  NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_investments_family ON investments (family_id, sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_investments_family_name
    ON investments (family_id, lower(btrim(name)));

CREATE TABLE IF NOT EXISTS installments (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id          UUID NOT NULL,
    name               TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    total_minor        BIGINT NOT NULL CHECK (total_minor > 0),
    monthly_minor      BIGINT NOT NULL CHECK (monthly_minor > 0),
    months             INT  NOT NULL CHECK (months BETWEEN 1 AND 120),
    currency_code      TEXT NOT NULL CHECK (length(btrim(currency_code)) = 3),
    account_id         UUID NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    category_id        UUID NOT NULL UNIQUE REFERENCES categories (id) ON DELETE RESTRICT,
    member_id          UUID NOT NULL,
    created_by_user_id UUID NOT NULL,
    purchased_on       DATE NOT NULL,
    day_of_month       INT  NOT NULL CHECK (day_of_month BETWEEN 1 AND 31),
    next_due_on        DATE NOT NULL,
    status             TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active','paid_off','cancelled')),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_installments_family ON installments (family_id, created_at);
CREATE INDEX IF NOT EXISTS idx_installments_due ON installments (status, next_due_on);
