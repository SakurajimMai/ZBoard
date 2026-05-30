# Zboard 文档中心

Zboard 是一个纯自研商业订阅节点中控系统。它由控制面（Go API Server）、节点 Agent、订阅生成、节点运行时管理、流量统计和后台管理组成，直接管理 Xray-core / sing-box core 运行时。

## 阅读顺序

1. [产品需求](./prd.md) — 产品定位、MVP 范围、用户角色和上线闭环
2. [系统架构](./architecture.md) — 控制面、数据面、服务边界和仓库结构
3. [数据库设计](./database.md) — 核心业务表、索引和幂等设计
4. [API 设计](./api.md) — 用户端、后台、Agent、订阅和支付回调接口
5. [Agent 协议](./agent-protocol.md) — 节点注册、心跳、拉任务、上报结果和签名规则
6. [订阅系统](./subscription.md) — 订阅格式、生成流程和异常返回策略
7. [节点运行时](./node-runtime.md) — Xray / sing-box 配置生成、校验、重载和流量采集
8. [安全设计](./security.md) — 后台、API、支付、Agent、订阅和基础设施安全
9. [部署方案](./deploy.md) — Docker 容器化、环境变量、备份和监控

## 快速开始

### 本地开发（API Server）

```bash
# 环境要求：Go 1.25+
# 最小 SQLite 配置
export ZBOARD_DB_DIALECT=sqlite
export ZBOARD_DB_PATH=./data/zboard.sqlite
export ZBOARD_ADMIN_SETUP_TOKEN=dev-admin-token
export ZBOARD_TOKEN_SECRET="$(openssl rand -hex 32)"

go run ./apps/api-server/cmd/server
# 健康检查: GET http://127.0.0.1:3000/health
```

### 本地开发（前端）

```bash
cd frontend
pnpm install
pnpm dev
# http://localhost:3000
```

### Docker 部署（推荐）

```bash
docker compose -f deploy/docker/docker-compose.prod.yml up -d
```

详见 [部署方案](./deploy.md)。

## 技术栈

| 组件 | 技术 |
|------|------|
| 控制面 API | Go 1.25 + Gin + sqlx |
| 数据库 | MySQL / MariaDB / PostgreSQL / SQLite（三方言 migration） |
| 前端 | Next.js 16 + React 19 + Tailwind 4 + shadcn/ui |
| Node Agent | Go（独立二进制，systemd / Docker） |
| 节点运行时 | Xray-core / sing-box core |
| CI/CD | GitHub Actions → DockerHub |
| 容器化 | 多阶段 Docker 构建（Alpine） |

## 支持的协议

| 协议 | 运行时 | 传输 | 安全 | 抗封锁能力 |
|------|--------|------|------|-----------|
| VLESS + Vision | Xray / sing-box | tcp / ws / grpc | tls / reality | ★★★★ |
| VMess | Xray / sing-box | tcp / ws / grpc | tls | ★★★ |
| Trojan | Xray / sing-box | tcp / ws / grpc | tls | ★★★ |
| SS-2022 | Xray / sing-box | tcp | tls / none | ★★★★ |
| Hysteria2 | sing-box | udp (QUIC) | tls | ★★★★★ |
| TUIC | sing-box | udp (QUIC) | tls | ★★★★★ |

## 核心原则

- Agent 主动连接中央系统，中央系统不主动 SSH 到节点
- Xray-core / sing-box 只作为运行时执行引擎，不作为管理面板
- MVP 优先跑通付款、开通、配置同步、订阅可用、流量统计、到期/超额停用链路
- 所有敏感数据（node_secret、reality_private_key）只在创建时返回一次，库里只存哈希或仅服务端使用
