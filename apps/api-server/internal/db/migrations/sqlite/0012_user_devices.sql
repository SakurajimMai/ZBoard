-- 0012 subscription device registry (SQLite)
CREATE TABLE IF NOT EXISTS user_devices (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  fingerprint TEXT NOT NULL,
  ip TEXT,
  user_agent TEXT,
  first_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
  last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
  UNIQUE(user_id, fingerprint)
);

CREATE INDEX IF NOT EXISTS idx_user_devices_user_seen ON user_devices(user_id, last_seen_at);
