-- Drop indexes
DROP INDEX IF EXISTS idx_investments_status;
DROP INDEX IF EXISTS idx_investments_user_id;
DROP INDEX IF EXISTS idx_recurring_next_execution;
DROP INDEX IF EXISTS idx_recurring_user_id;
DROP INDEX IF EXISTS idx_transactions_executed_at;
DROP INDEX IF EXISTS idx_transactions_wallet_id;
DROP INDEX IF EXISTS idx_wallets_user_id;

-- Drop tables in reverse order
DROP TABLE IF EXISTS investments;
DROP TABLE IF EXISTS recurring;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS wallets;
DROP TABLE IF EXISTS users;

