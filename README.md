# Zboard

自研商业订阅节点中控系统。直接管理 Xray-core / sing-box core 运行时，涵盖用户、订单、支付、套餐、订阅、节点、Agent、配置生成、流量统计和后台管理。

## 快速部署

```bash
# 控制面
mkdir /opt/zboard && cd /opt/zboard
curl -O https://raw.githubusercontent.com/SakurajimMai/ZBoard/main/deploy/docker/docker-compose.prod.yml
curl -O https://raw.githubusercontent.com/SakurajimMai/ZBoard/main/deploy/docker/api.env.example
cp api.env.example api.env
nano api.env  # 填入数据库连接和管理员信息
docker compose -f docker-compose.prod.yml up -d

# 节点
mkdir /opt/zboard-agent && cd /opt/zboard-agent
curl -O https://raw.githubusercontent.com/SakurajimMai/ZBoard/main/deploy/docker/docker-compose.agent.yml
curl -O https://raw.githubusercontent.com/SakurajimMai/ZBoard/main/deploy/docker/agent.env.example
cp agent.env.example agent.env
nano agent.env  # 填入控制面地址和节点密钥
docker compose -f docker-compose.agent.yml up -d
```

详细步骤见 [部署文档](./docs/deploy.md)。

## 技术栈

| 组件 | 技术 |
|------|------|
| API Server | Go 1.25 + Gin + sqlx |
| 数据库 | MySQL / MariaDB / PostgreSQL / SQLite |
| 前端 | Next.js 16 + React 19 + Tailwind 4 + shadcn/ui |
| Node Agent | Go（Docker / systemd） |
| 运行时 | Xray-core / sing-box core |
| CI/CD | GitHub Actions → DockerHub |

## 支持协议

| 协议 | 运行时 | 抗封锁 |
|------|--------|--------|
| VLESS + Vision + Reality | Xray / sing-box | ★★★★ |
| SS-2022 | Xray / sing-box | ★★★★ |
| Hysteria2（含端口跳跃） | sing-box | ★★★★★ |
| TUIC | sing-box | ★★★★★ |
| VMess / Trojan | Xray / sing-box | ★★★ |

## 支付渠道

通过管理后台动态配置，无需重启：

- **易支付** — 支付宝 / 微信
- **Creem** — 信用卡 / Apple Pay / Google Pay
- **NOWPayments** — BTC / ETH / USDT 加密货币

## Docker 镜像

| 镜像 | 说明 |
|------|------|
| `sakurajiamai/zboard-api` | API Server |
| `sakurajiamai/zboard-frontend` | 前端 |
| `sakurajiamai/zboard-agent` | Node Agent |

## 本地开发

```bash
# API Server（Go 1.25+）
export ZBOARD_DB_DIALECT=sqlite
export ZBOARD_ADMIN_EMAIL=admin@local.dev
export ZBOARD_ADMIN_PASSWORD=dev123
export ZBOARD_TOKEN_SECRET=dev-secret
go run ./apps/api-server/cmd/server

# 前端
cd frontend && pnpm install && pnpm dev
```

## 文档

- [部署方案](./docs/deploy.md) — Docker Compose 部署、环境变量、反向代理
- [API 设计](./docs/api.md) — 接口文档
- [系统架构](./docs/architecture.md) — 整体设计
- [Agent 协议](./docs/agent-protocol.md) — HMAC 签名、任务拉取
- [数据库设计](./docs/database.md) — 表结构

## License

Private — All rights reserved.
