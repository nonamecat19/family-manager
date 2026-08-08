-- Drop in reverse dependency order. All tables are new in 000001, so a CASCADE-free drop
-- works because the schema is empty before this migration applies.

DROP TABLE IF EXISTS meal_plan_entries;
DROP TABLE IF EXISTS recipe_comments;
DROP TABLE IF EXISTS recipe_favorites;
DROP TABLE IF EXISTS recipe_steps;
DROP TABLE IF EXISTS recipe_ingredients;
DROP TABLE IF EXISTS recipes;
DROP TABLE IF EXISTS recipe_subcategories;
DROP TABLE IF EXISTS recipe_categories;