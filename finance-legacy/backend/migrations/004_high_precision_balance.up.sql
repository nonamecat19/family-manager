-- Increase precision for wallet balance to support very small decimal values
-- NUMERIC(30, 18) allows up to 30 total digits with 18 decimal places
-- This is suitable for cryptocurrency and very precise financial calculations
ALTER TABLE wallets ALTER COLUMN balance TYPE NUMERIC(30, 18);



