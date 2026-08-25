-- Per-serving nutrition. The cookbook this schema was first filled from prints kcal and the
-- three macros on every page, and its sections are organised around them ("Білкові супи",
-- "Вегетаріанські білкові") — dropping the numbers on import would throw away the reason
-- those recipes were collected.
--
-- NOT NULL DEFAULT 0 rather than nullable, following `rating`: 0 reads as "not recorded",
-- and every query that sorts or filters on these avoids a NULLS LAST clause. A recipe with a
-- genuine 0 kcal does not exist.
--
-- REAL, not NUMERIC: these are printed approximations from a book (32 g protein, 18.5 g fat),
-- not money. Exact decimal arithmetic buys nothing and costs a pgtype.Numeric at every call
-- site.
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS kcal      INT  NOT NULL DEFAULT 0;
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS protein_g REAL NOT NULL DEFAULT 0;
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS fat_g     REAL NOT NULL DEFAULT 0;
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS carbs_g   REAL NOT NULL DEFAULT 0;

DO $$
BEGIN
    ALTER TABLE recipes ADD CONSTRAINT recipes_nutrition_nonneg
        CHECK (kcal >= 0 AND protein_g >= 0 AND fat_g >= 0 AND carbs_g >= 0);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;
