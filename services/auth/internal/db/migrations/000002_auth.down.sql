DROP TABLE IF EXISTS refresh_tokens;
DROP INDEX IF EXISTS idx_users_email_lower;
ALTER TABLE users RENAME COLUMN password_hash TO password;
