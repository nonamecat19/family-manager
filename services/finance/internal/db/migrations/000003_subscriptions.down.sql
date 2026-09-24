DROP TABLE IF EXISTS subscription_occurrences;
DROP TABLE IF EXISTS subscriptions;
UPDATE category_groups SET role = '' WHERE role = 'subscriptions';
ALTER TABLE category_groups DROP CONSTRAINT IF EXISTS category_groups_role_check;
ALTER TABLE category_groups ADD CONSTRAINT category_groups_role_check
  CHECK (role IN ('', 'investments', 'installments'));
