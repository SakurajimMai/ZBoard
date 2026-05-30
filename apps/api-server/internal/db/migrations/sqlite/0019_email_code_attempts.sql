-- 0019 email code brute-force guard (SQLite)
ALTER TABLE email_codes ADD COLUMN failed_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE email_codes ADD COLUMN locked_at DATETIME;
