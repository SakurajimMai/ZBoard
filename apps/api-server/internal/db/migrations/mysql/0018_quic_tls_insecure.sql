-- 0018 allow clients to skip verification for agent-generated QUIC certificates (MySQL / MariaDB)
ALTER TABLE nodes ADD COLUMN tls_insecure INT NOT NULL DEFAULT 1;
