-- Revert to original precision
ALTER TABLE wallets ALTER COLUMN balance TYPE NUMERIC(15, 2);



