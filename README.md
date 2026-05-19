# Zboard

Zboard 是一个纯自研商业订阅节点中控系统，目标是用自研控制面、自研 Node Agent、自研订阅生成和自研流量统计系统，直接管理节点运行时。

底层节点运行时可以使用 Xray-core 或 sing-box core，但它们只作为执行引擎，不作为管理面板。真正的用户、订单、支付、套餐、订阅、节点、Agent、配置生成、流量统计和后台管理都由 Zboard 自己实现。

## 核心原则

- 不依赖 3x-ui。
- 不兼容 3x-ui。
- 不做 3x-ui Adapter。
- 不读取 3x-ui 数据库。
- 不调用 3x-ui API。
- 不保留 3x-ui 迁移逻辑。
- Agent 主动连接中央系统，中央系统不主动 SSH 到节点。
- MVP 优先跑通商业闭环，不一开始追求大而全。

## 当前技术栈

- 控制面：Go 1.25 + Gin + sqlx
- 数据库：MySQL / MariaDB / PostgreSQL / SQLite 三方言通用，线上对接 MariaDB
- 前端：独立的 Next.js 16 + React 19 + Tailwind 4 + shadcn/ui 工程（`frontend/`）
- 部署：正式环境 Docker Compose，本地开发直接 `go run`
- Agent（计划中）：Go + systemd，HTTPS + HMAC

## 文档入口

[docs/README.md](./docs/README.md)。建议阅读顺序：

1. [产品需求](./docs/prd.md)
2. [系统架构](./docs/architecture.md)
3. [数据库设计](./docs/database.md)
4. [API 设计](./docs/api.md)
5. [Agent 协议](./docs/agent-protocol.md)
6. [订阅系统](./docs/subscription.md)
7. [节点运行时](./docs/node-runtime.md)
8. [安全设计](./docs/security.md)
9. [部署方案](./docs/deploy.md)

## 总体架构

```text
用户前台 (frontend/, Next.js)
  ↓
商业 API Server (apps/api-server, Go + Gin)
  ↓
订单 / 支付 / 套餐 / 用户 / 订阅 / 节点 / 任务 / 流量
  ↓
Agent Gateway (内置在 api-server，HMAC)
  ↓
自研 Node Agent (待实现)
  ↓
Xray-core / sing-box core
  ↓
VPS 节点
```

## MVP 闭环

```text
用户付款 → 系统开通套餐 → 生成订阅 token → 生成节点配置
        → Agent 同步配置 → 用户订阅可用 → Agent 上报流量 → 超额/到期自动停用
```

## 本地后端启动

环境要求：Go 1.25+。

仓库根目录：

```bash
go run ./apps/api-server/cmd/server
```

最小本地 SQLite 配置：

```env
ZBOARD_HOST=127.0.0.1
ZBOARD_PORT=3000
ZBOARD_DB_DIALECT=sqlite
ZBOARD_DB_PATH=./data/zboard.sqlite
ZBOARD_ADMIN_SETUP_TOKEN=dev-admin-token
ZBOARD_TOKEN_SECRET=dev-token-secret
```

MariaDB / MySQL 配置：

```env
ZBOARD_DB_DIALECT=mysql
ZBOARD_DB_DSN=user:pass@tcp(host:3306)/zboard?parseTime=true&charset=utf8mb4
```

PostgreSQL 配置：

```env
ZBOARD_DB_DIALECT=postgres
ZBOARD_DB_DSN=postgres://user:pass@host:5432/zboard?sslmode=require
```

健康检查：

```text
GET /health
```

## 主要接口

详见 [apps/api-server/README.md](./apps/api-server/README.md)。三类入口：

- `POST /api/v1/*`：用户端，Bearer 鉴权，订单 / 支付幂等。
- `POST /api/admin/v1/*`：后台 Bearer，bootstrap 后登录。
- `POST /api/agent/v1/*`：Agent，强制 HMAC 五头 + nonce 防重放。
- `GET  /api/sub/:token?target=base64|clash|sing-box`：订阅渲染。

## 正式环境容器化

```bash
docker compose -f deploy/docker/docker-compose.prod.yml up -d --build
```

构建为静态 Go 二进制，运行时镜像 `alpine:3.20`。MariaDB / Postgres 由外部托管，不在 compose 里启动。本地开发不需要执行 Docker。

## 下一步

1. PostgreSQL 在真库上跑通完整测试（migration / 业务 / agent 流）。
2. Node Agent 实现（Go，systemd，HMAC）。
3. 节点 path / sni / Reality 等字段补 schema + 渲染器。
4. Worker cron 化。
5. Go 单测覆盖（store / authsvc / bizsvc / subrender / runtime / worker）。
