-- 0019 email code brute-force guard (MySQL / MariaDB)
ALTER TABLE email_codes ADD COLUMN failed_attempts INT NOT NULL DEFAULT 0;
ALTER TABLE email_codes ADD COLUMN locked_at DATETIME NULL;
