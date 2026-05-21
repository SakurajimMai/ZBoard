-- 0012 subscription device registry (PostgreSQL)
CREATE TABLE IF NOT EXISTS user_devices (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  fingerprint TEXT NOT NULL,
  ip TEXT,
  user_agent TEXT,
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, fingerprint)
);

CREATE INDEX IF NOT EXISTS idx_user_devices_user_seen ON user_devices(user_id, last_seen_at);
