-- 0019 email code brute-force guard (PostgreSQL)
ALTER TABLE email_codes ADD COLUMN IF NOT EXISTS failed_attempts INT NOT NULL DEFAULT 0;
ALTER TABLE email_codes ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ;
