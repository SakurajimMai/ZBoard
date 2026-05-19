-- Initial schema (PostgreSQL)
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  balance NUMERIC(18,2) NOT NULL DEFAULT 0,
  plan_id BIGINT,
  expired_at TIMESTAMPTZ,
  traffic_limit BIGINT NOT NULL DEFAULT 0,
  traffic_used BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS plans (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  price NUMERIC(18,2) NOT NULL,
  duration_days INT NOT NULL,
  traffic_limit BIGINT NOT NULL,
  device_limit INT NOT NULL DEFAULT 3,
  speed_limit INT NOT NULL DEFAULT 0,
  node_group_id BIGINT,
  status TEXT NOT NULL DEFAULT 'active',
  sort INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
  id BIGSERIAL PRIMARY KEY,
  order_no TEXT NOT NULL UNIQUE,
  user_id BIGINT NOT NULL,
  plan_id BIGINT NOT NULL,
  amount NUMERIC(18,2) NOT NULL,
  currency TEXT NOT NULL DEFAULT 'CNY',
  status TEXT NOT NULL DEFAULT 'pending',
  paid_at TIMESTAMPTZ,
  cancelled_at TIMESTAMPTZ,
  expired_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

CREATE TABLE IF NOT EXISTS payments (
  id BIGSERIAL PRIMARY KEY,
  payment_no TEXT NOT NULL UNIQUE,
  order_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  provider TEXT NOT NULL,
  provider_trade_no TEXT,
  amount NUMERIC(18,2) NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  paid_at TIMESTAMPTZ,
  raw_payload TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_payments_order ON payments(order_id);

CREATE TABLE IF NOT EXISTS payment_callbacks (
  id BIGSERIAL PRIMARY KEY,
  provider TEXT NOT NULL,
  provider_event_id TEXT,
  order_no TEXT,
  signature_valid SMALLINT NOT NULL DEFAULT 1,
  processed SMALLINT NOT NULL DEFAULT 0,
  processed_at TIMESTAMPTZ,
  raw_headers TEXT,
  raw_body TEXT,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (provider, provider_event_id)
);

CREATE TABLE IF NOT EXISTS subscription_tokens (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  token TEXT NOT NULL UNIQUE,
  token_hash TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'active',
  last_access_ip TEXT,
  last_access_user_agent TEXT,
  last_access_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS nodes (
  id BIGSERIAL PRIMARY KEY,
  node_code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  region TEXT,
  host TEXT NOT NULL,
  port INT NOT NULL DEFAULT 443,
  protocol TEXT NOT NULL DEFAULT 'vless',
  transport TEXT NOT NULL DEFAULT 'tcp',
  security TEXT NOT NULL DEFAULT 'tls',
  runtime_type TEXT NOT NULL DEFAULT 'xray',
  agent_version TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  last_heartbeat_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS node_agents (
  id BIGSERIAL PRIMARY KEY,
  node_id BIGINT NOT NULL UNIQUE,
  node_secret_hash TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  version TEXT,
  os_info TEXT,
  runtime_info TEXT,
  registered_at TIMESTAMPTZ,
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS agent_heartbeats (
  id BIGSERIAL PRIMARY KEY,
  node_id BIGINT NOT NULL,
  agent_version TEXT,
  runtime_status TEXT,
  runtime_info TEXT,
  system_load TEXT,
  reported_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_hb_node_time ON agent_heartbeats(node_id, reported_at);

CREATE TABLE IF NOT EXISTS admin_users (
  id BIGSERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'admin',
  two_factor_enabled SMALLINT NOT NULL DEFAULT 0,
  two_factor_secret TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS admin_sessions (
  id BIGSERIAL PRIMARY KEY,
  admin_id BIGINT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_sessions (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS node_groups (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  sort INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS node_users (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  node_id BIGINT NOT NULL,
  client_id TEXT NOT NULL,
  protocol TEXT NOT NULL,
  enabled SMALLINT NOT NULL DEFAULT 1,
  upload BIGINT NOT NULL DEFAULT 0,
  download BIGINT NOT NULL DEFAULT 0,
  speed_limit INT NOT NULL DEFAULT 0,
  device_limit INT NOT NULL DEFAULT 0,
  last_sync_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, node_id)
);

CREATE TABLE IF NOT EXISTS runtime_configs (
  id BIGSERIAL PRIMARY KEY,
  node_id BIGINT NOT NULL,
  version TEXT NOT NULL,
  config_hash TEXT NOT NULL,
  config_json TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  applied_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (node_id, version)
);

CREATE TABLE IF NOT EXISTS node_tasks (
  id BIGSERIAL PRIMARY KEY,
  task_id TEXT NOT NULL UNIQUE,
  node_id BIGINT NOT NULL,
  task_type TEXT NOT NULL,
  payload TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  retry_count INT NOT NULL DEFAULT 0,
  max_retry_count INT NOT NULL DEFAULT 5,
  locked_at TIMESTAMPTZ,
  executed_at TIMESTAMPTZ,
  failed_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tasks_node_status ON node_tasks(node_id, status);

CREATE TABLE IF NOT EXISTS node_task_logs (
  id BIGSERIAL PRIMARY KEY,
  task_id TEXT NOT NULL,
  node_id BIGINT NOT NULL,
  status TEXT NOT NULL,
  message TEXT,
  detail TEXT,
  reported_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_task_logs_task ON node_task_logs(task_id);

CREATE TABLE IF NOT EXISTS traffic_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  node_id BIGINT NOT NULL,
  upload_delta BIGINT NOT NULL DEFAULT 0,
  download_delta BIGINT NOT NULL DEFAULT 0,
  total_delta BIGINT NOT NULL DEFAULT 0,
  reported_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_traffic_user_time ON traffic_logs(user_id, reported_at);

CREATE TABLE IF NOT EXISTS user_traffic_snapshots (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL UNIQUE,
  upload_total BIGINT NOT NULL DEFAULT 0,
  download_total BIGINT NOT NULL DEFAULT 0,
  total_used BIGINT NOT NULL DEFAULT 0,
  traffic_limit BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subscription_access_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT,
  token_hash TEXT,
  target TEXT,
  ip TEXT,
  user_agent TEXT,
  result TEXT NOT NULL,
  reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  actor_type TEXT NOT NULL,
  actor_id BIGINT,
  action TEXT NOT NULL,
  resource_type TEXT,
  resource_id TEXT,
  ip TEXT,
  user_agent TEXT,
  detail TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_type, actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);

CREATE TABLE IF NOT EXISTS idempotency_keys (
  id BIGSERIAL PRIMARY KEY,
  key_value TEXT NOT NULL UNIQUE,
  scope TEXT NOT NULL,
  request_hash TEXT,
  response_body TEXT,
  status TEXT NOT NULL DEFAULT 'processing',
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS agent_nonces (
  id BIGSERIAL PRIMARY KEY,
  node_id BIGINT NOT NULL,
  nonce TEXT NOT NULL,
  ts BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (node_id, nonce)
);
