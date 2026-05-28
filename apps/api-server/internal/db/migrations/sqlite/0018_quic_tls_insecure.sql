-- 0018 allow clients to skip verification for agent-generated QUIC certificates (SQLite)
ALTER TABLE nodes ADD COLUMN tls_insecure INTEGER NOT NULL DEFAULT 1;
