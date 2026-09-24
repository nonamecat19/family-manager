ALTER TABLE category_groups DROP CONSTRAINT IF EXISTS category_groups_role_check;
ALTER TABLE category_groups ADD CONSTRAINT category_groups_role_check
  CHECK (role IN ('', 'investments', 'installments', 'subscriptions'));

CREATE TABLE IF NOT EXISTS subscriptions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id          UUID NOT NULL,
    name               TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    amount_minor       BIGINT NOT NULL CHECK (amount_minor > 0),
    currency_code      TEXT NOT NULL CHECK (length(btrim(currency_code)) = 3),
    type               TEXT NOT NULL DEFAULT 'expense' CHECK (type IN ('expense','income','transfer')),
    category_id        UUID NOT NULL UNIQUE REFERENCES categories (id) ON DELETE RESTRICT,
    account_id         UUID NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    member_id          UUID NOT NULL,
    created_by_user_id UUID NOT NULL,
    interval_count     INT  NOT NULL DEFAULT 1 CHECK (interval_count >= 1),
    interval_unit      TEXT NOT NULL DEFAULT 'month' CHECK (interval_unit IN ('day','week','month','year')),
    day_of_month       INT  NOT NULL DEFAULT 1 CHECK (day_of_month BETWEEN 1 AND 31),
    day_of_week        TEXT NOT NULL DEFAULT '' CHECK (day_of_week IN ('','monday','tuesday','wednesday','thursday','friday','saturday','sunday')),
    next_due_on        DATE NOT NULL,
    end_on             DATE,
    last_posted_on     DATE,
    auto_post          BOOLEAN NOT NULL DEFAULT FALSE,
    active             BOOLEAN NOT NULL DEFAULT TRUE,
    status             TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','cancelled')),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_family ON subscriptions (family_id, created_at);
CREATE INDEX IF NOT EXISTS idx_subscriptions_due ON subscriptions (status, active, next_due_on);

CREATE TABLE IF NOT EXISTS subscription_occurrences (
    subscription_id UUID NOT NULL REFERENCES subscriptions (id) ON DELETE CASCADE,
    due_on DATE NOT NULL,
    PRIMARY KEY (subscription_id, due_on)
);
