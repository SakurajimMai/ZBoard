-- 0012 subscription device registry (MySQL/MariaDB)
CREATE TABLE IF NOT EXISTS user_devices (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  fingerprint VARCHAR(128) NOT NULL,
  ip VARCHAR(64) NULL,
  user_agent VARCHAR(512) NULL,
  first_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_device_fingerprint (user_id, fingerprint),
  KEY idx_user_devices_user_seen (user_id, last_seen_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
