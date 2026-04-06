-- Currencies table
CREATE TABLE currencies (
    id SERIAL PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    type TEXT CHECK (type IN ('fiat', 'crypto')) NOT NULL,
    exchange_rate NUMERIC(15, 8) NOT NULL DEFAULT 1.0,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert default currencies
-- Note: Exchange rates are stored in database and should be updated regularly via API
INSERT INTO currencies (code, type, exchange_rate, name) VALUES
    ('USD', 'fiat', 1.0, 'US Dollar'),
    ('UAH', 'fiat', 0.027, 'Ukrainian Hryvnia'), -- 1 USD = ~37 UAH (approximate, update via API)
    ('USDT', 'crypto', 1.0, 'Tether'),
    ('USDC', 'crypto', 1.0, 'USD Coin'),
    ('ETH', 'crypto', 2500.0, 'Ethereum'), -- 1 USD = ~0.0004 ETH (approximate, update via API)
    ('BTC', 'crypto', 45000.0, 'Bitcoin'); -- 1 USD = ~0.000022 BTC (approximate, update via API)

-- Create index for faster lookups
CREATE INDEX idx_currencies_code ON currencies(code);
CREATE INDEX idx_currencies_type ON currencies(type);

