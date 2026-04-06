-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Wallets table
CREATE TABLE wallets (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT CHECK (type IN ('cash', 'credit_card', 'crypto')) NOT NULL,
    currency TEXT NOT NULL,
    balance NUMERIC(15, 2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Transactions table
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    wallet_id INT REFERENCES wallets(id) ON DELETE CASCADE,
    type TEXT CHECK (type IN ('income', 'expense', 'transfer')) NOT NULL,
    amount NUMERIC(15, 2) NOT NULL,
    currency TEXT NOT NULL,
    description TEXT,
    category TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    executed_at TIMESTAMP
);

-- Recurring payments/income table
CREATE TABLE recurring (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    wallet_id INT REFERENCES wallets(id) ON DELETE CASCADE,
    type TEXT CHECK (type IN ('income', 'expense')) NOT NULL,
    amount NUMERIC(15, 2) NOT NULL,
    currency TEXT NOT NULL,
    description TEXT,
    recurrence TEXT CHECK (recurrence IN ('daily', 'weekly', 'monthly')) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    next_execution TIMESTAMP
);

-- Investments table
CREATE TABLE investments (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    wallet_id INT REFERENCES wallets(id) ON DELETE CASCADE,
    type TEXT CHECK (type IN ('crypto_stake', 'other')) NOT NULL,
    coin TEXT,
    amount NUMERIC(15, 2),
    start_date DATE NOT NULL,
    end_date DATE,
    daily_reward NUMERIC(15, 2),
    status TEXT CHECK (status IN ('active', 'completed')) NOT NULL DEFAULT 'active'
);

-- Indexes for better query performance
CREATE INDEX idx_wallets_user_id ON wallets(user_id);
CREATE INDEX idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX idx_transactions_executed_at ON transactions(executed_at);
CREATE INDEX idx_recurring_user_id ON recurring(user_id);
CREATE INDEX idx_recurring_next_execution ON recurring(next_execution);
CREATE INDEX idx_investments_user_id ON investments(user_id);
CREATE INDEX idx_investments_status ON investments(status);

