# Zboard 部署方案

## 架构概览

```
┌─────────────────────────────────────────────────────┐
│                    用户设备                           │
│         (Clash Meta / sing-box / V2rayN)            │
└──────────────────────┬──────────────────────────────┘
                       │ 订阅链接 / 代理连接
                       ▼
┌─────────────────────────────────────────────────────┐
│              控制面服务器                              │
│  ┌──────────────┐  ┌──────────────┐                 │
│  │ zboard-api   │  │zboard-frontend│                │
│  │ (Go, :3000)  │  │(Next.js,:3001)│                │
│  └──────┬───────┘  └──────────────┘                 │
│         │                                           │
│  ┌──────┴───────┐                                   │
│  │  MariaDB     │                                   │
│  │  (外部托管)   │                                   │
│  └──────────────┘                                   │
└─────────────────────────────────────────────────────┘
                       │ HMAC 签名 API
                       ▼
┌─────────────────────────────────────────────────────┐
│              VPS 节点 (每台)                          │
│  ┌──────────────┐  ┌──────────────┐                 │
│  │ zboard-agent │──│ Xray/sing-box│                 │
│  │ (Go)         │  │ (子进程)      │                 │
│  └──────────────┘  └──────────────┘                 │
└─────────────────────────────────────────────────────┘
```

## Docker 镜像

| 镜像 | 说明 |
|------|------|
| `sakurajiamai/zboard-api:latest` | Go API Server |
| `sakurajiamai/zboard-frontend:latest` | Next.js 前端 |
| `sakurajiamai/zboard-agent:latest` | Node Agent |

镜像由 GitHub Actions 在每次 push 到 `main` 时自动构建并推送到 DockerHub。
镜像发布为多架构 manifest，支持 `linux/amd64` 与 `linux/arm64`；ARM 服务器无需更换镜像名，Docker 会按宿主机架构自动拉取对应版本。

---

## 一、控制面部署

### 前置条件

- 一台服务器（1C1G 起步）
- Docker + Docker Compose 已安装
- MariaDB / MySQL 数据库（可用云数据库）
- 域名（可选，用于 HTTPS）

### 步骤

#### 1. 创建部署目录

```bash
mkdir -p /opt/zboard && cd /opt/zboard
```

#### 2. 下载 compose 文件

```bash
curl -O https://raw.githubusercontent.com/SakurajimMai/ZBoard/main/deploy/docker/docker-compose.prod.yml
curl -O https://raw.githubusercontent.com/SakurajimMai/ZBoard/main/deploy/docker/api.env.example
cp api.env.example api.env
```

#### 3. 编辑环境变量

```bash
nano api.env
```

必填项：

```env
ZBOARD_DB_DIALECT=mysql
ZBOARD_DB_DSN=user:password@tcp(db-host:3306)/zboard?parseTime=true&charset=utf8mb4
ZBOARD_ADMIN_SETUP_TOKEN=你的随机初始化密钥
ZBOARD_TOKEN_SECRET=你的随机签名密钥
```

支付渠道（按需启用）：

```env
# 易支付
ZBOARD_EPAY_API_URL=https://pay.example.com
ZBOARD_EPAY_PID=商户ID
ZBOARD_EPAY_KEY=密钥

# Creem（海外卡）
ZBOARD_CREEM_API_KEY=your_api_key
ZBOARD_CREEM_WEBHOOK_SECRET=your_webhook_secret

# NOWPayments（加密货币）
ZBOARD_NOWPAY_API_KEY=your_api_key
ZBOARD_NOWPAY_IPN_SECRET=your_ipn_secret
```

#### 4. 启动服务

如果前端和 API 暴露在同一台服务器的默认端口，可直接留空 `NEXT_PUBLIC_API_URL`；前端会按浏览器当前主机推导 `http(s)://<host>:3000`。如果 API 使用其它公网地址或端口，请在启动前设置为浏览器可访问的地址：

```env
NEXT_PUBLIC_API_URL=http://你的服务器IP:3000
```

注意：不要填写 Docker 内部服务名，例如 `http://api:3000`，用户浏览器无法解析这个地址。

```bash
docker compose -f docker-compose.prod.yml up -d
```

#### 5. 检查状态

```bash
# 健康检查
curl http://127.0.0.1:3000/health
# 应返回 {"status":"ok"}

# 查看日志
docker compose -f docker-compose.prod.yml logs -f
```

#### 6. 初始化管理员

```bash
curl -X POST http://127.0.0.1:3000/api/admin/v1/auth/bootstrap \
  -H "Content-Type: application/json" \
  -d '{
    "setup_token": "你在api.env里设置的ZBOARD_ADMIN_SETUP_TOKEN",
    "email": "admin@example.com",
    "password": "你的强密码"
  }'
```

初始化成功后 `ZBOARD_ADMIN_SETUP_TOKEN` 即失效，无法再次使用。

### 更新控制面

```bash
cd /opt/zboard
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

---

## 二、节点部署

### 前置条件

- VPS（任意 Linux）
- Docker + Docker Compose 已安装
- Xray 或 sing-box 已安装
- 控制面已部署且可访问

### 步骤

#### 1. 安装运行时

```bash
# Xray（推荐）
bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install

# 或 sing-box
bash -c "$(curl -fsSL https://sing-box.app/deb-install.sh)"
```

#### 2. 在控制面创建节点

通过管理后台或 API 创建节点：

```bash
curl -X POST https://panel.example.com/api/admin/v1/nodes \
  -H "Authorization: Bearer 你的管理员token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "HK-01",
    "region": "香港",
    "host": "节点IP或域名",
    "port": 443,
    "protocol": "vless",
    "transport": "tcp",
    "security": "reality",
    "fingerprint": "chrome",
    "reality_public_key": "你的公钥",
    "reality_private_key": "你的私钥",
    "reality_short_id": "短ID",
    "reality_server_name": "www.cloudflare.com",
    "reality_dest": "www.cloudflare.com:443"
  }'
```

返回值中的 `node_id` 和 `node_secret` **只显示一次**，务必记录。

#### 3. 创建部署目录

```bash
mkdir -p /root/docker/agent && cd /root/docker/agent
```

#### 4. 下载 compose 文件

```bash
curl -O https://raw.githubusercontent.com/SakurajimMai/ZBoard/main/deploy/docker/docker-compose.agent.yml
mv docker-compose.agent.yml docker-compose.yml   # 让 docker compose 直接识别
curl -O https://raw.githubusercontent.com/SakurajimMai/ZBoard/main/deploy/docker/agent.env.example
cp agent.env.example agent.env                   # 必须与 docker-compose.yml 同目录
mkdir -p data/etc data/agent
```

> ⚠️ `agent.env` 必须放在 `docker-compose.yml` 同目录,不要放进 `data/` 子目录,否则启动时会报 `env file ... not found`。

#### 5. 编辑 agent.env

```bash
nano agent.env
```

```env
ZBOARD_AGENT_API_BASE_URL=https://panel.example.com
ZBOARD_AGENT_NODE_ID=创建节点返回的ID
ZBOARD_AGENT_NODE_SECRET=创建节点返回的密钥
ZBOARD_AGENT_RUNTIME_BINARY=/usr/local/bin/xray
ZBOARD_AGENT_RUNTIME_TYPE=xray
ZBOARD_AGENT_STATS_API_ADDR=127.0.0.1:10085
```

#### 6. 启动 Agent

```bash
docker compose up -d
```

#### 7. 验证

```bash
# 查看日志
docker compose -f docker-compose.agent.yml logs -f

# 应看到：
# agent registered with control plane (node_id=X)
# task ... sync_config applied (changed=true)
```

#### 8. 生成节点配置

在控制面触发配置同步：

```bash
curl -X POST https://panel.example.com/api/admin/v1/nodes/节点ID/sync-config \
  -H "Authorization: Bearer 管理员token"
```

Agent 会在下次 pull（默认 10 秒）时自动拉取并应用配置。

### 更新 Agent

```bash
cd /opt/zboard-agent
docker compose -f docker-compose.agent.yml pull
docker compose -f docker-compose.agent.yml up -d
```

---

## 三、支付配置

### 易支付 (EasyPay)

适用场景：国内用户，支付宝/微信支付。

1. 注册易支付商户，获取 `PID` 和 `Key`
2. 在易支付后台设置异步通知地址：`https://panel.example.com/api/v1/payments/epay/callback`
3. 配置环境变量：

```env
ZBOARD_EPAY_API_URL=https://你的易支付地址
ZBOARD_EPAY_PID=商户ID
ZBOARD_EPAY_KEY=商户密钥
```

用户发起支付：

```
POST /api/v1/orders/:order_no/pay?provider=epay&pay_type=alipay
POST /api/v1/orders/:order_no/pay?provider=epay&pay_type=wxpay
```

### Creem

适用场景：海外用户，信用卡/Apple Pay/Google Pay。

1. 注册 Creem 账号，获取 API Key 和 Webhook Secret
2. 在 Creem 后台配置 Webhook URL：`https://panel.example.com/api/v1/payments/creem/callback`
3. 配置环境变量：

```env
ZBOARD_CREEM_API_KEY=你的API密钥
ZBOARD_CREEM_WEBHOOK_SECRET=你的Webhook签名密钥
```

用户发起支付：

```
POST /api/v1/orders/:order_no/pay?provider=creem
```

### NOWPayments（加密货币）

适用场景：匿名支付，BTC/ETH/USDT。

1. 注册 NOWPayments，获取 API Key 和 IPN Secret
2. 在 NOWPayments 后台设置 IPN URL：`https://panel.example.com/api/v1/payments/nowpayments/callback`
3. 配置环境变量：

```env
ZBOARD_NOWPAY_API_KEY=你的API密钥
ZBOARD_NOWPAY_IPN_SECRET=你的IPN签名密钥
```

用户发起支付：

```
POST /api/v1/orders/:order_no/pay?provider=nowpayments&pay_type=usdttrc20
POST /api/v1/orders/:order_no/pay?provider=nowpayments&pay_type=btc
POST /api/v1/orders/:order_no/pay?provider=nowpayments&pay_type=eth
```

---

## 四、环境变量参考

### API Server

| 变量 | 必填 | 说明 |
|------|------|------|
| `ZBOARD_HOST` | 否 | 监听地址，默认 `127.0.0.1` |
| `ZBOARD_PORT` | 否 | 监听端口，默认 `3000` |
| `ZBOARD_DB_DIALECT` | 是 | `mysql` / `postgres` / `sqlite` |
| `ZBOARD_DB_DSN` | 是 | 数据库连接串 |
| `ZBOARD_ADMIN_SETUP_TOKEN` | 是 | 首次初始化管理员密钥 |
| `ZBOARD_TOKEN_SECRET` | 是 | 会话签名密钥 |
| `ZBOARD_CORS_ORIGINS` | 否 | 允许的前端域名（逗号分隔，`*` 放行全部）；独立 API 域名部署时必填前端域名 |
| `ZBOARD_TRUSTED_PROXIES` | 否 | 受信任反代/CDN 的 CIDR 列表（逗号分隔）。空 = 不信任任何代理，限流按直连 IP。反代/CDN 后填这一层网段，使限流按真实客户端 IP 计数 |
| `ZBOARD_TRUSTED_PLATFORM` | 否 | CDN 真实客户端 IP 头别名：`cloudflare`/`cf`→`CF-Connecting-IP`、`google`/`gae`、`fly`/`flyio`，或直接写头名。与 `ZBOARD_TRUSTED_PROXIES` 二选一，用 CDN 时推荐。**仅当源站只接受该 CDN 回源时安全** |
| `ZBOARD_EPAY_*` | 否 | 易支付配置 |
| `ZBOARD_CREEM_*` | 否 | Creem 配置 |
| `ZBOARD_NOWPAY_*` | 否 | NOWPayments 配置 |
| `ZBOARD_SMTP_*` | 否 | SMTP 邮件配置（HOST/PORT/USER/PASS/FROM）；留空则验证码仅打印日志 |

> ⚠️ 支付回调依赖后台「站点设置 → site_url」配置一个公网可达的 origin。未配置时
> `POST /api/v1/orders/:order_no/pay` 返回 `503 site_url_unconfigured` 拒绝发起支付，
> 避免网关收款后回调指向 `127.0.0.1` 而无法激活套餐。过期订单在发起支付前即被
> `409 order_expired` 拒绝。详见 [安全设计](./security.md)。

### Node Agent

| 变量 | 必填 | 说明 |
|------|------|------|
| `ZBOARD_AGENT_API_BASE_URL` | 是 | 控制面地址 |
| `ZBOARD_AGENT_NODE_ID` | 是 | 节点 ID |
| `ZBOARD_AGENT_NODE_SECRET` | 是 | 节点密钥（创建时返回一次） |
| `ZBOARD_AGENT_RUNTIME_BINARY` | 否 | 运行时路径，默认 `/usr/local/bin/xray` |
| `ZBOARD_AGENT_RUNTIME_TYPE` | 否 | `xray` 或 `sing-box`，默认 `xray` |
| `ZBOARD_AGENT_STATS_API_ADDR` | 否 | stats gRPC 地址，默认 `127.0.0.1:10085` |
| `ZBOARD_AGENT_HEARTBEAT_INTERVAL` | 否 | 心跳间隔，默认 `30s` |
| `ZBOARD_AGENT_PULL_INTERVAL` | 否 | 任务拉取间隔，默认 `10s` |
| `ZBOARD_AGENT_TRAFFIC_INTERVAL` | 否 | 流量上报间隔，默认 `60s` |

---

## 五、运维

### 数据库

启动时自动执行 migration，无需手动建表。支持 MySQL/MariaDB、PostgreSQL、SQLite。

### 监控

- 健康检查：`GET /health`
- Agent 心跳：管理后台可查看 `last_heartbeat_at`
- 审计日志：`GET /api/admin/v1/audit-logs`

### 备份

MariaDB 建议使用云厂商自动备份或 `mysqldump` 定时任务。

### 更新

```bash
# 控制面
docker compose -f docker-compose.prod.yml pull && docker compose -f docker-compose.prod.yml up -d

# 节点
docker compose -f docker-compose.agent.yml pull && docker compose -f docker-compose.agent.yml up -d
```

### 日志

```bash
# 控制面
docker compose -f docker-compose.prod.yml logs -f api
docker compose -f docker-compose.prod.yml logs -f frontend

# 节点
docker compose -f docker-compose.agent.yml logs -f agent
```
