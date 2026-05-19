# Zboard API Server (Go)

Zboard 后端控制面，纯 Go 实现，使用 Gin + sqlx + 三方言 SQL migration（MySQL / PostgreSQL / SQLite）。线上目标数据库是 MariaDB（走 MySQL 协议）。

## 已实现能力

商业主链路：注册 / 登录 / 订阅 token / 套餐 CRUD / 订单 / 模拟支付 / 支付幂等（`Idempotency-Key`）/ 回调去重。

订阅渲染：Clash Meta / sing-box / Base64 共享同一份归一化节点字段，输出协议、地址、端口、UUID、传输和 TLS 映射；同名节点自动追加序号去重。

节点与 Agent：管理员创建节点（返回一次性 `node_secret`，库里只存 `sha256` 哈希），生成 Xray / sing-box 最小可用 runtime config 和同步任务，回滚通过反向同步任务实现。Agent 主动连入：`/api/agent/v1/*` 写接口必须携带 HMAC 五头（`X-Zboard-Node-Id` / `Timestamp` / `Nonce` / `Body-SHA256` / `Signature`），timestamp 5 分钟窗口，nonce 写 `agent_nonces` 防重放。

Worker 维护：到期用户禁用、超额用户禁用、每节点生成 `disable_user` 任务、超时 `running` 任务清理、失败任务按 `max_retry_count` 自动重新入队。

后台：管理员 bootstrap（首个管理员需 `ZBOARD_ADMIN_SETUP_TOKEN`）、Bearer 会话、用户 / 套餐 / 订单 / 支付 / 节点 / 任务 / 流量 / 审计列表。

## 启动

仓库根目录执行：

```bash
go run ./apps/api-server/cmd/server
```

或在 `apps/api-server` 下：

```bash
go run ./cmd/server
```

环境变量见 `deploy/env/api.env.example`。最小集合：

```env
ZBOARD_DB_DIALECT=mysql
ZBOARD_DB_DSN=user:pass@tcp(host:3306)/zboard?parseTime=true&charset=utf8mb4
ZBOARD_ADMIN_SETUP_TOKEN=...
ZBOARD_TOKEN_SECRET=...
```

SQLite 本地最简：

```env
ZBOARD_DB_DIALECT=sqlite
ZBOARD_DB_PATH=./data/zboard.sqlite
```

健康检查：`GET /health`，前端入口 `/app` 与 `/admin` 由 `frontend/` Next.js 工程负责（独立部署）。

## 数据库 migration

启动时自动运行。SQL 文件按方言分目录：

```
internal/db/migrations/{mysql,postgres,sqlite}/NNNN_*.sql
```

新表 / 列：在所有三个方言下补同名 SQL，`schema_migrations` 按文件名记录已应用版本。

## 主要接口

```text
GET  /health
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/me
POST /api/v1/auth/logout
GET  /api/v1/plans
POST /api/v1/orders                       (Idempotency-Key)
POST /api/v1/orders/:order_no/pay         (Idempotency-Key)
POST /api/v1/payments/mock-callback
GET  /api/v1/subscription
POST /api/v1/subscription/reset-token
GET  /api/sub/:token?target=base64|clash|sing-box

POST /api/admin/v1/auth/bootstrap
POST /api/admin/v1/auth/login
GET  /api/admin/v1/auth/me
POST /api/admin/v1/auth/logout
GET  /api/admin/v1/users
POST /api/admin/v1/users/:id/disable
POST /api/admin/v1/users/:id/enable
GET  /api/admin/v1/plans
POST /api/admin/v1/plans
GET  /api/admin/v1/orders
GET  /api/admin/v1/payments
GET  /api/admin/v1/payment-callbacks
GET  /api/admin/v1/nodes
POST /api/admin/v1/nodes
POST /api/admin/v1/nodes/:id/sync-config
GET  /api/admin/v1/nodes/:id/runtime-configs
POST /api/admin/v1/runtime-configs/:version/rollback
GET  /api/admin/v1/node-tasks
GET  /api/admin/v1/traffic/users
GET  /api/admin/v1/traffic/logs
GET  /api/admin/v1/audit-logs
POST /api/admin/v1/workers/maintenance/run

POST /api/agent/v1/register      (HMAC)
POST /api/agent/v1/heartbeat     (HMAC)
POST /api/agent/v1/tasks/pull    (HMAC)
POST /api/agent/v1/tasks/:task_id/result   (HMAC)
POST /api/agent/v1/traffic/report          (HMAC)
```

后台首次初始化：

```text
POST /api/admin/v1/auth/bootstrap         # 用 ZBOARD_ADMIN_SETUP_TOKEN 创建 owner
POST /api/admin/v1/auth/login              # 拿 Bearer token
Authorization: Bearer <token>              # 访问 /api/admin/v1/*
```

Agent 接口拒绝请求体明文 `node_secret`。所有 `/api/agent/v1/*` 写接口必须携带：

```text
X-Zboard-Node-Id
X-Zboard-Timestamp        (5 分钟窗口)
X-Zboard-Nonce
X-Zboard-Body-SHA256
X-Zboard-Signature        (HMAC-SHA256, key = sha256(node_secret) hex)
```

## 测试脚本

`scripts/agent_smoke.go` 是 Agent HMAC 流程的本地烟测；`scripts/age_user.go` 用于在测试库里直接把用户过期时间改到过去，验证 worker 维护任务。两者都带 `//go:build ignore`，`go run` 单独执行：

```bash
go run ./scripts/agent_smoke.go -base http://127.0.0.1:3000 -node 1 -secret <plaintext>
```

## 正式环境容器化

```bash
docker compose -f deploy/docker/docker-compose.prod.yml up -d --build
```

镜像基于 `golang:1.25-alpine` 构建，运行时镜像 `alpine:3.20`，纯静态二进制 `zboard-api`。MariaDB / Postgres 由外部托管，本地开发不需要执行 Docker。
