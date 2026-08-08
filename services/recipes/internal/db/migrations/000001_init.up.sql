-- services/recipes owns the family cookbook. Every row is scoped by family_id, which comes
-- from the access token's family_id claim (stamped by services/auth from services/family) —
-- there is NO foreign key to the family service's tables, by design: crossing a service
-- boundary in SQL is what the contract in libs/proto exists to prevent.
--
-- user_id columns likewise reference users in services/auth and carry no FK.

CREATE TABLE IF NOT EXISTS recipe_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id   UUID NOT NULL,
    name        TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recipe_categories_family
    ON recipe_categories (family_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_recipe_categories_family_name
    ON recipe_categories (family_id, lower(btrim(name)));

CREATE TABLE IF NOT EXISTS recipe_subcategories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES recipe_categories (id) ON DELETE CASCADE,
    family_id   UUID NOT NULL,
    name        TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recipe_subcategories_category
    ON recipe_subcategories (category_id);
CREATE INDEX IF NOT EXISTS idx_recipe_subcategories_family
    ON recipe_subcategories (family_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_recipe_subcategories_category_name
    ON recipe_subcategories (category_id, lower(btrim(name)));

CREATE TABLE IF NOT EXISTS recipes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL,
    title           TEXT NOT NULL CHECK (length(btrim(title)) > 0),
    description     TEXT NOT NULL DEFAULT '',
    category_id     UUID REFERENCES recipe_categories (id) ON DELETE SET NULL,
    subcategory_id  UUID REFERENCES recipe_subcategories (id) ON DELETE SET NULL,
    servings        INT  NOT NULL DEFAULT 1 CHECK (servings > 0),
    prep_seconds    INT  NOT NULL DEFAULT 0 CHECK (prep_seconds >= 0),
    cook_seconds    INT  NOT NULL DEFAULT 0 CHECK (cook_seconds >= 0),
    author_user_id  UUID NOT NULL,
    favorite_count  INT  NOT NULL DEFAULT 0,
    comment_count   INT  NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recipes_family       ON recipes (family_id);
CREATE INDEX IF NOT EXISTS idx_recipes_family_cat   ON recipes (family_id, category_id);
CREATE INDEX IF NOT EXISTS idx_recipes_family_subcat ON recipes (family_id, subcategory_id);
CREATE INDEX IF NOT EXISTS idx_recipes_family_title ON recipes (family_id, lower(btrim(title)));

-- Ingredients and steps are owned by the recipe and cascade with it. They are stored as rows,
-- not a JSON column, so TotalIngredients can SUM them server-side across the meal plan.
CREATE TABLE IF NOT EXISTS recipe_ingredients (
    recipe_id   UUID NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    position    INT  NOT NULL DEFAULT 0,
    name        TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    amount      TEXT NOT NULL DEFAULT '',
    unit        TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (recipe_id, position)
);

CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_name_unit
    ON recipe_ingredients (lower(btrim(name)), lower(btrim(unit)));

CREATE TABLE IF NOT EXISTS recipe_steps (
    recipe_id         UUID NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    position          INT  NOT NULL,
    instruction       TEXT NOT NULL CHECK (length(btrim(instruction)) > 0),
    duration_seconds  INT  NOT NULL DEFAULT 0 CHECK (duration_seconds >= 0),
    PRIMARY KEY (recipe_id, position)
);

CREATE TABLE IF NOT EXISTS recipe_favorites (
    recipe_id   UUID NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (recipe_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_recipe_favorites_user ON recipe_favorites (user_id);

CREATE TABLE IF NOT EXISTS recipe_comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id   UUID NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    body        TEXT NOT NULL CHECK (length(btrim(body)) > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recipe_comments_recipe
    ON recipe_comments (recipe_id, created_at);

-- Meal plan: a family assigns recipes to calendar days + slots.
CREATE TABLE IF NOT EXISTS meal_plan_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id   UUID NOT NULL,
    recipe_id   UUID NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    -- date is a calendar day, not a timestamptz: a meal plan is about days, not instants.
    plan_date   DATE NOT NULL,
    slot        TEXT NOT NULL CHECK (slot IN ('breakfast','lunch','dinner','snack','dessert')),
    servings    INT  NOT NULL DEFAULT 0 CHECK (servings >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- One recipe per slot per day per family — adding the same soup to lunch twice is a UI
    -- mistake, not a second helping.
    UNIQUE (family_id, plan_date, slot)
);

CREATE INDEX IF NOT EXISTS idx_meal_plan_family_date
    ON meal_plan_entries (family_id, plan_date);