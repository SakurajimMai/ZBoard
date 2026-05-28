-- 0018 allow clients to skip verification for agent-generated QUIC certificates (PostgreSQL)
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS tls_insecure INTEGER NOT NULL DEFAULT 1;
