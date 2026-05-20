-- 0005 payment_providers table (PostgreSQL)
CREATE TABLE IF NOT EXISTS payment_providers (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL DEFAULT '',
  provider_type TEXT NOT NULL,
  config_json TEXT NOT NULL DEFAULT '{}',
  enabled SMALLINT NOT NULL DEFAULT 1,
  sort INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
