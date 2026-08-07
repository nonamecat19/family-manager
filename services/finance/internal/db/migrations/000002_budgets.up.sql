-- A budget is a recurring limit: "500 a month from the 15th", not "500 between two dates".
-- The window is derived from period + start_on at read time, so a budget never needs a row
-- per month and never goes stale when nobody opens the app.
CREATE TABLE IF NOT EXISTS budgets (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id    UUID NOT NULL,
    name         TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    -- NULL covers every expense in the household. Keeping the total budget in the same table
    -- as per-category ones means one progress query shape instead of two.
    category_id  UUID REFERENCES categories (id) ON DELETE CASCADE,
    limit_minor  BIGINT NOT NULL CHECK (limit_minor > 0),
    currency_code TEXT NOT NULL CHECK (currency_code ~ '^[A-Z]{3}$'),
    period       TEXT NOT NULL CHECK (period IN ('week', 'month', 'year')),
    start_on     DATE NOT NULL,
    archived     BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_budgets_family ON budgets (family_id, archived, sort_order);

-- One budget per category per period. Two overlapping limits on the same spending would both
-- be "the" budget, and no screen could say which one was exceeded.
-- Two partial indexes, because NULL never equals NULL in a unique index: without the second
-- one a household could hold any number of total budgets.
CREATE UNIQUE INDEX IF NOT EXISTS idx_budgets_one_per_category
    ON budgets (family_id, category_id, period)
    WHERE archived = FALSE AND category_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_budgets_one_total
    ON budgets (family_id, period)
    WHERE archived = FALSE AND category_id IS NULL;
