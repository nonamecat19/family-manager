-- Rating is a single family verdict, not a per-user score: this cookbook is shared and the
-- question people actually ask is "is this one good", not "who liked it". 0 means unrated,
-- which is why the column is NOT NULL DEFAULT 0 rather than nullable — sorting by rating
-- then puts unrated recipes last without a NULLS LAST clause on every query.
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS rating SMALLINT NOT NULL DEFAULT 0;

DO $$
BEGIN
    ALTER TABLE recipes ADD CONSTRAINT recipes_rating_range CHECK (rating BETWEEN 0 AND 5);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

-- notes is the free-form margin of a recipe ("double the garlic", "grandma's version").
-- Kept separate from description: description sells the dish in list views, notes are the
-- cook's private amendments and are only shown on the detail screen.
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';

-- Sorting by rating and by total time is server-side (see ListRecipes), so both need an
-- index scoped the same way every recipe query is: by family.
CREATE INDEX IF NOT EXISTS idx_recipes_family_rating ON recipes (family_id, rating DESC);
CREATE INDEX IF NOT EXISTS idx_recipes_family_time
    ON recipes (family_id, ((prep_seconds + cook_seconds)));
