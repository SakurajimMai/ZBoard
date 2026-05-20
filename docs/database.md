# 数据库设计文档

## 设计原则

- PostgreSQL 作为主业务数据库。
- 所有金额字段使用 `NUMERIC(12,2)`，不使用浮点数。
- 所有流量字段使用 `BIGINT`，单位为字节。
- 所有外部回调、节点任务和敏感操作必须可追溯。
- 支付回调、节点任务和 Agent 上报必须支持幂等。
- 业务状态字段先使用 `VARCHAR`，后续可按需要迁移为枚举类型。

## 迁移管理

当前本地测试环境使用 SQLite3，迁移文件位于：

```text
apps/api-server/internal/db/migrations/sqlite
```

API Server 启动时会：

1. 创建 `schema_migrations` 表。
2. 按文件名版本顺序读取 `*.sql`。
3. 跳过已记录版本。
4. 在事务中执行未应用 migration。
5. 写入 `schema_migrations`。

PostgreSQL 迁移预留目录位于：

```text
apps/api-server/internal/db/migrations/postgres
```

迁移命名规则：

```text
0001_initial_schema.sql
0002_add_xxx.sql
```

SQLite 与 PostgreSQL 的版本号应保持语义一致，但 SQL 方言可以不同。

## 核心实体关系

```text
users
  ├── orders
  ├── subscription_tokens
  ├── node_users
  └── user_traffic_snapshots

plans
  ├── users
  └── orders

nodes
  ├── node_agents
  ├── node_tasks
  ├── node_users
  ├── runtime_configs
  └── traffic_logs
```

## 用户表

```sql
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  balance NUMERIC(12,2) DEFAULT 0 NOT NULL,
  plan_id BIGINT,
  expired_at TIMESTAMP,
  traffic_limit BIGINT DEFAULT 0 NOT NULL,
  traffic_used BIGINT DEFAULT 0 NOT NULL,
  status VARCHAR(32) DEFAULT 'active' NOT NULL,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_users_plan_id ON users(plan_id);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_expired_at ON users(expired_at);
```

## 套餐表

```sql
CREATE TABLE plans (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  price NUMERIC(12,2) NOT NULL,
  duration_days INT NOT NULL,
  traffic_limit BIGINT NOT NULL,
  device_limit INT DEFAULT 3 NOT NULL,
  speed_limit INT DEFAULT 0 NOT NULL,
  node_group_id BIGINT,
  status VARCHAR(32) DEFAULT 'active' NOT NULL,
  sort INT DEFAULT 0 NOT NULL,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_plans_status_sort ON plans(status, sort);
```

## 订单表

```sql
CREATE TABLE orders (
  id BIGSERIAL PRIMARY KEY,
  order_no VARCHAR(64) UNIQUE NOT NULL,
  user_id BIGINT NOT NULL,
  plan_id BIGINT NOT NULL,
  amount NUMERIC(12,2) NOT NULL,
  currency VARCHAR(16) DEFAULT 'CNY' NOT NULL,
  status VARCHAR(32) DEFAULT 'pending' NOT NULL,
  paid_at TIMESTAMP,
  cancelled_at TIMESTAMP,
  expired_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_plan_id ON orders(plan_id);
CREATE INDEX idx_orders_status ON orders(status);
```

订单状态建议：

```text
pending
paid
cancelled
expired
refunded
```

## 支付记录表

```sql
CREATE TABLE payments (
  id BIGSERIAL PRIMARY KEY,
  payment_no VARCHAR(64) UNIQUE NOT NULL,
  order_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  provider VARCHAR(64) NOT NULL,
  provider_trade_no VARCHAR(128),
  amount NUMERIC(12,2) NOT NULL,
  status VARCHAR(32) DEFAULT 'pending' NOT NULL,
  paid_at TIMESTAMP,
  raw_payload JSONB,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_provider_trade_no ON payments(provider_trade_no);
```

## 支付回调表

```sql
CREATE TABLE payment_callbacks (
  id BIGSERIAL PRIMARY KEY,
  provider VARCHAR(64) NOT NULL,
  provider_event_id VARCHAR(128),
  order_no VARCHAR(64),
  signature_valid BOOLEAN DEFAULT FALSE NOT NULL,
  processed BOOLEAN DEFAULT FALSE NOT NULL,
  processed_at TIMESTAMP,
  raw_headers JSONB,
  raw_body JSONB,
  error_message TEXT,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  UNIQUE(provider, provider_event_id)
);

CREATE INDEX idx_payment_callbacks_order_no ON payment_callbacks(order_no);
CREATE INDEX idx_payment_callbacks_processed ON payment_callbacks(processed);
```

## 节点组表

```sql
CREATE TABLE node_groups (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  description TEXT,
  status VARCHAR(32) DEFAULT 'active' NOT NULL,
  sort INT DEFAULT 0 NOT NULL,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);
```

## 节点表

```sql
CREATE TABLE nodes (
  id BIGSERIAL PRIMARY KEY,
  node_code VARCHAR(64) UNIQUE NOT NULL,
  name VARCHAR(100) NOT NULL,
  region VARCHAR(100),
  host VARCHAR(255) NOT NULL,
  public_ip VARCHAR(64),
  node_group_id BIGINT,
  traffic_rate NUMERIC(6,2) DEFAULT 1 NOT NULL,
  runtime_type VARCHAR(32) DEFAULT 'xray' NOT NULL,
  agent_version VARCHAR(64),
  status VARCHAR(32) DEFAULT 'active' NOT NULL,
  max_users INT DEFAULT 0 NOT NULL,
  max_bandwidth BIGINT DEFAULT 0 NOT NULL,
  last_heartbeat_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_nodes_node_group_id ON nodes(node_group_id);
CREATE INDEX idx_nodes_status ON nodes(status);
CREATE INDEX idx_nodes_last_heartbeat_at ON nodes(last_heartbeat_at);
```

`runtime_type` 只允许表达执行引擎：

```text
xray
singbox
custom
```

## Agent 表

```sql
CREATE TABLE node_agents (
  id BIGSERIAL PRIMARY KEY,
  node_id BIGINT UNIQUE NOT NULL,
  node_secret_hash TEXT NOT NULL,
  status VARCHAR(32) DEFAULT 'pending' NOT NULL,
  version VARCHAR(64),
  os_info JSONB,
  runtime_info JSONB,
  registered_at TIMESTAMP,
  last_seen_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_node_agents_status ON node_agents(status);
CREATE INDEX idx_node_agents_last_seen_at ON node_agents(last_seen_at);
```

## Agent 心跳表

```sql
CREATE TABLE agent_heartbeats (
  id BIGSERIAL PRIMARY KEY,
  node_id BIGINT NOT NULL,
  agent_version VARCHAR(64),
  runtime_status VARCHAR(32),
  runtime_info JSONB,
  system_load JSONB,
  reported_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_agent_heartbeats_node_reported ON agent_heartbeats(node_id, reported_at DESC);
```

## 节点用户表

```sql
CREATE TABLE node_users (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  node_id BIGINT NOT NULL,
  client_id VARCHAR(128) NOT NULL,
  protocol VARCHAR(50) NOT NULL,
  enabled BOOLEAN DEFAULT TRUE NOT NULL,
  upload BIGINT DEFAULT 0 NOT NULL,
  download BIGINT DEFAULT 0 NOT NULL,
  speed_limit INT DEFAULT 0 NOT NULL,
  device_limit INT DEFAULT 0 NOT NULL,
  last_sync_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL,
  UNIQUE(user_id, node_id)
);

CREATE INDEX idx_node_users_node_id ON node_users(node_id);
CREATE INDEX idx_node_users_enabled ON node_users(enabled);
CREATE INDEX idx_node_users_client_id ON node_users(client_id);
```

## 运行配置表

```sql
CREATE TABLE runtime_configs (
  id BIGSERIAL PRIMARY KEY,
  node_id BIGINT NOT NULL,
  version VARCHAR(64) NOT NULL,
  config_hash VARCHAR(128) NOT NULL,
  config_json JSONB NOT NULL,
  status VARCHAR(32) DEFAULT 'pending' NOT NULL,
  applied_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  UNIQUE(node_id, version)
);

CREATE INDEX idx_runtime_configs_node_status ON runtime_configs(node_id, status);
```

## 节点任务表

```sql
CREATE TABLE node_tasks (
  id BIGSERIAL PRIMARY KEY,
  task_id VARCHAR(64) UNIQUE NOT NULL,
  node_id BIGINT NOT NULL,
  task_type VARCHAR(64) NOT NULL,
  payload JSONB NOT NULL,
  status VARCHAR(32) DEFAULT 'pending' NOT NULL,
  retry_count INT DEFAULT 0 NOT NULL,
  max_retry_count INT DEFAULT 5 NOT NULL,
  locked_at TIMESTAMP,
  executed_at TIMESTAMP,
  failed_reason TEXT,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_node_tasks_node_status ON node_tasks(node_id, status);
CREATE INDEX idx_node_tasks_task_type ON node_tasks(task_type);
```

MVP 任务类型：

```text
sync_full_config
disable_user
delete_user
reload_runtime
```

后续任务类型：

```text
create_user
enable_user
update_user_limit
restart_runtime
upgrade_agent
rotate_node_secret
```

## 节点任务日志表

```sql
CREATE TABLE node_task_logs (
  id BIGSERIAL PRIMARY KEY,
  task_id VARCHAR(64) NOT NULL,
  node_id BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL,
  message TEXT,
  detail JSONB,
  reported_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_node_task_logs_task_id ON node_task_logs(task_id);
CREATE INDEX idx_node_task_logs_node_id ON node_task_logs(node_id);
```

## 流量日志表

```sql
CREATE TABLE traffic_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  node_id BIGINT NOT NULL,
  upload_delta BIGINT DEFAULT 0 NOT NULL,
  download_delta BIGINT DEFAULT 0 NOT NULL,
  total_delta BIGINT DEFAULT 0 NOT NULL,
  reported_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_traffic_logs_user_reported ON traffic_logs(user_id, reported_at DESC);
CREATE INDEX idx_traffic_logs_node_reported ON traffic_logs(node_id, reported_at DESC);
```

## 用户流量快照表

```sql
CREATE TABLE user_traffic_snapshots (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT UNIQUE NOT NULL,
  upload_total BIGINT DEFAULT 0 NOT NULL,
  download_total BIGINT DEFAULT 0 NOT NULL,
  total_used BIGINT DEFAULT 0 NOT NULL,
  traffic_limit BIGINT DEFAULT 0 NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);
```

## 订阅 token 表

```sql
CREATE TABLE subscription_tokens (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  token_hash VARCHAR(128) UNIQUE NOT NULL,
  status VARCHAR(32) DEFAULT 'active' NOT NULL,
  last_access_ip VARCHAR(64),
  last_access_user_agent TEXT,
  last_access_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_subscription_tokens_user_id ON subscription_tokens(user_id);
CREATE INDEX idx_subscription_tokens_status ON subscription_tokens(status);
```

## 订阅访问日志表

```sql
CREATE TABLE subscription_access_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT,
  token_hash VARCHAR(128),
  target VARCHAR(32),
  ip VARCHAR(64),
  user_agent TEXT,
  result VARCHAR(32) NOT NULL,
  reason TEXT,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_subscription_access_logs_user_created ON subscription_access_logs(user_id, created_at DESC);
CREATE INDEX idx_subscription_access_logs_token_created ON subscription_access_logs(token_hash, created_at DESC);
```

## 管理员表

```sql
CREATE TABLE admin_users (
  id BIGSERIAL PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role VARCHAR(64) DEFAULT 'admin' NOT NULL,
  two_factor_enabled BOOLEAN DEFAULT FALSE NOT NULL,
  two_factor_secret TEXT,
  status VARCHAR(32) DEFAULT 'active' NOT NULL,
  last_login_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);
```

## 审计日志表

```sql
CREATE TABLE audit_logs (
  id BIGSERIAL PRIMARY KEY,
  actor_type VARCHAR(32) NOT NULL,
  actor_id BIGINT,
  action VARCHAR(128) NOT NULL,
  resource_type VARCHAR(64),
  resource_id VARCHAR(64),
  ip VARCHAR(64),
  user_agent TEXT,
  detail JSONB,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_type, actor_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
```

## 系统配置表

```sql
CREATE TABLE system_settings (
  id BIGSERIAL PRIMARY KEY,
  key VARCHAR(128) UNIQUE NOT NULL,
  value JSONB NOT NULL,
  description TEXT,
  updated_by BIGINT,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);
```

## 幂等键表

```sql
CREATE TABLE idempotency_keys (
  id BIGSERIAL PRIMARY KEY,
  key VARCHAR(128) UNIQUE NOT NULL,
  scope VARCHAR(64) NOT NULL,
  request_hash VARCHAR(128),
  response_body JSONB,
  status VARCHAR(32) DEFAULT 'processing' NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_idempotency_keys_scope ON idempotency_keys(scope);
CREATE INDEX idx_idempotency_keys_expires_at ON idempotency_keys(expires_at);
```

## 后续优化

- 为高频流量表按月分区。
- 为 `traffic_logs` 和 `agent_heartbeats` 设置归档策略。
- 为 `audit_logs` 设置只追加策略，避免后台误改。
- 对 `subscription_tokens.token_hash` 存 hash，不保存明文 token。
- 对 `node_secret` 只保存 hash，不保存明文密钥。
