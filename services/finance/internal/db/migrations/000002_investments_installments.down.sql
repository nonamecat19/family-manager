DROP TABLE IF EXISTS installments;
DROP TABLE IF EXISTS investments;
DROP INDEX IF EXISTS idx_category_groups_family_role;
ALTER TABLE category_groups DROP COLUMN IF EXISTS role;
