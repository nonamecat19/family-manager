DROP INDEX IF EXISTS idx_recipes_family_time;
DROP INDEX IF EXISTS idx_recipes_family_rating;
ALTER TABLE recipes DROP CONSTRAINT IF EXISTS recipes_rating_range;
ALTER TABLE recipes DROP COLUMN IF EXISTS notes;
ALTER TABLE recipes DROP COLUMN IF EXISTS rating;
