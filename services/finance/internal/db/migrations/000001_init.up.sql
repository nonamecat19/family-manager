CREATE TABLE IF NOT EXISTS finance_settings (
    family_id                       UUID PRIMARY KEY,
    base_currency_code              TEXT NOT NULL CHECK (length(btrim(base_currency_code)) = 3),
    timezone                        TEXT NOT NULL DEFAULT 'UTC',
    week_starts_on                  TEXT NOT NULL DEFAULT 'monday'
        CHECK (week_starts_on IN ('monday','tuesday','wednesday','thursday','friday','saturday','sunday')),
    overspend_notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    pin_lock_enabled                BOOLEAN NOT NULL DEFAULT FALSE,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS finance_members (
    family_id         UUID NOT NULL,
    user_id           UUID NOT NULL,
    display_name      TEXT NOT NULL DEFAULT '',
    initial           TEXT NOT NULL DEFAULT '',
    avatar_color_step INT  NOT NULL DEFAULT 0 CHECK (avatar_color_step BETWEEN 0 AND 7),
    role              TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner','member')),
    status            TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','pending')),
    email             TEXT NOT NULL DEFAULT '',
    joined_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (family_id, user_id)
);

CREATE TABLE IF NOT EXISTS accounts (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id                  UUID NOT NULL,
    name                       TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    kind                       TEXT NOT NULL DEFAULT 'cash'
        CHECK (kind IN ('cash','card','bank','savings','crypto','debt')),
    visibility                 TEXT NOT NULL DEFAULT 'shared'
        CHECK (visibility IN ('shared','private')),
    owner_member_id            UUID,
    currency_code              TEXT NOT NULL CHECK (length(btrim(currency_code)) = 3),
    opening_balance_minor      BIGINT NOT NULL DEFAULT 0,
    icon                       TEXT NOT NULL DEFAULT '',
    color_step                 INT  NOT NULL DEFAULT 0 CHECK (color_step BETWEEN 0 AND 7),
    excluded_from_family_total BOOLEAN NOT NULL DEFAULT FALSE,
    archived                   BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order                 INT  NOT NULL DEFAULT 0,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT accounts_private_has_owner CHECK (
        (visibility = 'private' AND owner_member_id IS NOT NULL)
        OR (visibility = 'shared' AND owner_member_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_accounts_family ON accounts (family_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_accounts_family_visibility
    ON accounts (family_id, visibility, owner_member_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_family_name
    ON accounts (family_id, lower(btrim(name)));

CREATE TABLE IF NOT EXISTS category_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id  UUID NOT NULL,
    name       TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    kind       TEXT NOT NULL DEFAULT 'expense' CHECK (kind IN ('expense','income')),
    icon       TEXT NOT NULL DEFAULT '',
    color_step INT  NOT NULL DEFAULT 0 CHECK (color_step BETWEEN 0 AND 7),
    sort_order INT  NOT NULL DEFAULT 0,
    archived   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_category_groups_family
    ON category_groups (family_id, kind, sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_category_groups_family_name
    ON category_groups (family_id, kind, lower(btrim(name)));

CREATE TABLE IF NOT EXISTS categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id  UUID NOT NULL,
    group_id   UUID NOT NULL REFERENCES category_groups (id) ON DELETE CASCADE,
    name       TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    kind       TEXT NOT NULL DEFAULT 'expense' CHECK (kind IN ('expense','income')),
    icon       TEXT NOT NULL DEFAULT '',
    sort_order INT  NOT NULL DEFAULT 0,
    archived   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_categories_family_group
    ON categories (family_id, group_id, sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_group_name
    ON categories (group_id, lower(btrim(name)));

CREATE TABLE IF NOT EXISTS transactions (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id              UUID NOT NULL,
    type                   TEXT NOT NULL CHECK (type IN ('expense','income','transfer')),
    account_id             UUID NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    counter_account_id     UUID REFERENCES accounts (id) ON DELETE RESTRICT,
    category_id            UUID REFERENCES categories (id) ON DELETE SET NULL,
    amount_minor           BIGINT NOT NULL CHECK (amount_minor >= 0),
    currency_code          TEXT NOT NULL CHECK (length(btrim(currency_code)) = 3),
    received_amount_minor  BIGINT,
    received_currency_code TEXT NOT NULL DEFAULT '',
    note                   TEXT NOT NULL DEFAULT '',
    merchant               TEXT NOT NULL DEFAULT '',
    occurred_on            DATE NOT NULL,
    member_id              UUID NOT NULL,
    created_by_user_id     UUID NOT NULL,
    template_id            UUID,
    recurring_id           UUID,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT transactions_transfer_has_counter CHECK (
        (type = 'transfer' AND counter_account_id IS NOT NULL)
        OR (type <> 'transfer' AND counter_account_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_transactions_feed
    ON transactions (family_id, occurred_on DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_category
    ON transactions (family_id, category_id, occurred_on);
CREATE INDEX IF NOT EXISTS idx_transactions_member
    ON transactions (family_id, member_id, occurred_on);
CREATE INDEX IF NOT EXISTS idx_transactions_account
    ON transactions (family_id, account_id, occurred_on);
CREATE INDEX IF NOT EXISTS idx_transactions_counter_account
    ON transactions (counter_account_id) WHERE counter_account_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS budgets (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id        UUID NOT NULL,
    target_kind      TEXT NOT NULL CHECK (target_kind IN ('group','category')),
    group_id         UUID REFERENCES category_groups (id) ON DELETE CASCADE,
    category_id      UUID REFERENCES categories (id) ON DELETE CASCADE,
    limit_minor      BIGINT NOT NULL CHECK (limit_minor > 0),
    currency_code    TEXT NOT NULL CHECK (length(btrim(currency_code)) = 3),
    period           TEXT NOT NULL DEFAULT 'month' CHECK (period IN ('week','month','year')),
    start_on         DATE NOT NULL,
    member_id        UUID,
    notify_on_exceed BOOLEAN NOT NULL DEFAULT TRUE,
    archived         BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order       INT  NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT budgets_exactly_one_target CHECK (
        (target_kind = 'group'    AND group_id IS NOT NULL AND category_id IS NULL)
        OR (target_kind = 'category' AND category_id IS NOT NULL AND group_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_budgets_family ON budgets (family_id, sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_budgets_group_member
    ON budgets (family_id, group_id, COALESCE(member_id, '00000000-0000-0000-0000-000000000000'::uuid))
    WHERE target_kind = 'group' AND NOT archived;
CREATE UNIQUE INDEX IF NOT EXISTS idx_budgets_category_member
    ON budgets (family_id, category_id, COALESCE(member_id, '00000000-0000-0000-0000-000000000000'::uuid))
    WHERE target_kind = 'category' AND NOT archived;

CREATE TABLE IF NOT EXISTS quick_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id     UUID NOT NULL,
    owner_user_id UUID NOT NULL,
    label         TEXT NOT NULL CHECK (length(btrim(label)) > 0),
    icon          TEXT NOT NULL DEFAULT '',
    amount_minor  BIGINT NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
    currency_code TEXT NOT NULL CHECK (length(btrim(currency_code)) = 3),
    type          TEXT NOT NULL DEFAULT 'expense' CHECK (type IN ('expense','income')),
    category_id   UUID REFERENCES categories (id) ON DELETE SET NULL,
    account_id    UUID REFERENCES accounts (id) ON DELETE CASCADE,
    member_id     UUID NOT NULL,
    sort_order    INT  NOT NULL DEFAULT 0,
    usage_count   INT  NOT NULL DEFAULT 0,
    last_used_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quick_templates_owner
    ON quick_templates (family_id, owner_user_id, sort_order);

CREATE TABLE IF NOT EXISTS recurring_payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL,
    name            TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    amount_minor    BIGINT NOT NULL CHECK (amount_minor >= 0),
    currency_code   TEXT NOT NULL CHECK (length(btrim(currency_code)) = 3),
    type            TEXT NOT NULL DEFAULT 'expense' CHECK (type IN ('expense','income')),
    category_id     UUID REFERENCES categories (id) ON DELETE SET NULL,
    account_id      UUID REFERENCES accounts (id) ON DELETE CASCADE,
    member_id       UUID NOT NULL,
    interval_count  INT  NOT NULL DEFAULT 1 CHECK (interval_count > 0),
    interval_unit   TEXT NOT NULL DEFAULT 'month' CHECK (interval_unit IN ('day','week','month','year')),
    day_of_month    INT  NOT NULL DEFAULT 0 CHECK (day_of_month BETWEEN 0 AND 31),
    day_of_week     TEXT NOT NULL DEFAULT ''
        CHECK (day_of_week IN ('','monday','tuesday','wednesday','thursday','friday','saturday','sunday')),
    next_due_on     DATE NOT NULL,
    end_on          DATE,
    auto_post       BOOLEAN NOT NULL DEFAULT FALSE,
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    last_posted_on  DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recurring_payments_due
    ON recurring_payments (family_id, active, next_due_on);

CREATE TABLE IF NOT EXISTS reminders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    kind            TEXT NOT NULL DEFAULT 'custom'
        CHECK (kind IN ('budget_exceeded','recurring_due','custom')),
    title           TEXT NOT NULL DEFAULT '',
    due_at          TIMESTAMPTZ,
    repeat_interval INT  NOT NULL DEFAULT 0 CHECK (repeat_interval >= 0),
    repeat_unit     TEXT NOT NULL DEFAULT ''
        CHECK (repeat_unit IN ('','day','week','month','year')),
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reminders_user ON reminders (family_id, user_id, enabled);

CREATE TABLE IF NOT EXISTS widget_instances (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id          UUID NOT NULL,
    user_id            UUID NOT NULL,
    type               TEXT NOT NULL CHECK (type IN
        ('quick_add','month','category','budgets_and_family','recent_transactions','accounts')),
    size               TEXT NOT NULL DEFAULT '4x2'
        CHECK (size IN ('4x2','2x2','2x1','4x3','4x1')),
    scope_kind         TEXT NOT NULL DEFAULT 'family'
        CHECK (scope_kind IN ('family','member','account')),
    scope_member_id    UUID,
    scope_account_id   UUID,
    target_ref         UUID,
    target_account_ids UUID[] NOT NULL DEFAULT '{}',
    sort_order         INT  NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_widget_instances_user
    ON widget_instances (family_id, user_id, sort_order);
