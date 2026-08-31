-- Drop in reverse dependency order. All tables are new in 000001, so a CASCADE-free drop
-- works because the schema is empty before this migration applies.

DROP TABLE IF EXISTS widget_instances;
DROP TABLE IF EXISTS reminders;
DROP TABLE IF EXISTS recurring_payments;
DROP TABLE IF EXISTS quick_templates;
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS category_groups;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS finance_members;
DROP TABLE IF EXISTS finance_settings;
