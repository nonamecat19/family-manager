ALTER TABLE recipes ADD COLUMN IF NOT EXISTS rating SMALLINT NOT NULL DEFAULT 0;

DO $$
BEGIN
    ALTER TABLE recipes ADD CONSTRAINT recipes_rating_range CHECK (rating BETWEEN 0 AND 5);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

ALTER TABLE recipes ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_recipes_family_rating ON recipes (family_id, rating DESC);
CREATE INDEX IF NOT EXISTS idx_recipes_family_time
    ON recipes (family_id, ((prep_seconds + cook_seconds)));
