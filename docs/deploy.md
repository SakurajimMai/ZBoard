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
│              Docker Host (控制面)                     │
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

| 镜像 | 说明 | DockerHub |
|------|------|-----------|
| `sakurajiamai/zboard-api` | Go API Server | [链接](https://hub.docker.com/r/sakurajiamai/zboard-api) |
| `sakurajiamai/zboard-frontend` | Next.js 前端 | [链接](https://hub.docker.com/r/sakurajiamai/zboard-frontend) |
| `sakurajiamai/zboard-agent` | Node Agent | [链接](https://hub.docker.com/r/sakurajiamai/zboard-agent) |

镜像由 GitHub Actions 在每次 push 到 `main` 时自动构建并推送。

## 控制面部署

### 1. 准备环境变量

创建 `api.env`：

```env
ZBOARD_HOST=0.0.0.0
ZBOARD_PORT=3000
ZBOARD_DB_DIALECT=mysql
ZBOARD_DB_DSN=user:pass@tcp(your-mariadb-host:3306)/zboard?parseTime=true&charset=utf8mb4
ZBOARD_ADMIN_SETUP_TOKEN=你的一次性初始化密钥
ZBOARD_TOKEN_SECRET=随机生成的长字符串
```

### 2. Docker Compose 部署

```yaml
# docker-compose.yml
services:
  api:
    image: sakurajiamai/zboard-api:latest
    container_name: zboard-api
    env_file: ./api.env
    ports:
      - "3000:3000"
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://127.0.0.1:3000/health"]
      interval: 30s
      timeout: 5s
      retries: 3
    restart: unless-stopped

  frontend:
    image: sakurajiamai/zboard-frontend:latest
    container_name: zboard-frontend
    environment:
      - NEXT_PUBLIC_API_URL=http://api:3000
    ports:
      - "3001:3000"
    depends_on:
      api:
        condition: service_healthy
    restart: unless-stopped
```

```bash
docker compose up -d
```

### 3. 初始化管理员

```bash
curl -X POST http://your-server:3000/api/admin/v1/auth/bootstrap \
  -H "Content-Type: application/json" \
  -d '{"setup_token":"你的一次性初始化密钥","email":"admin@example.com","password":"强密码"}'
```

初始化完成后，`ZBOARD_ADMIN_SETUP_TOKEN` 即失效（admin_users 非空时 bootstrap 返回 409）。

### 4. 反向代理（可选）

如果需要 HTTPS，在前面加一层 Nginx / Caddy：

```nginx
server {
    listen 443 ssl http2;
    server_name panel.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /api/ {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location / {
        proxy_pass http://127.0.0.1:3001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 节点部署

### 方式一：Docker（推荐）

```bash
docker run -d \
  --name zboard-agent \
  --restart unless-stopped \
  --network host \
  -v /etc/zboard-agent:/etc/zboard-agent \
  -v /var/lib/zboard-agent:/var/lib/zboard-agent \
  -v /usr/local/bin/xray:/usr/local/bin/xray \
  sakurajiamai/zboard-agent:latest
```

配置文件 `/etc/zboard-agent/agent.env`：

```env
ZBOARD_AGENT_API_BASE_URL=https://panel.example.com
ZBOARD_AGENT_NODE_ID=1
ZBOARD_AGENT_NODE_SECRET=创建节点时返回的一次性密钥
ZBOARD_AGENT_RUNTIME_BINARY=/usr/local/bin/xray
ZBOARD_AGENT_RUNTIME_TYPE=xray
ZBOARD_AGENT_STATS_API_ADDR=127.0.0.1:10085
```

### 方式二：systemd（裸机）

```bash
wget -O /usr/local/bin/zboard-agent <release-url>
chmod +x /usr/local/bin/zboard-agent
mkdir -p /etc/zboard-agent /var/lib/zboard-agent

cat > /etc/systemd/system/zboard-agent.service << 'EOF'
[Unit]
Description=Zboard Node Agent
After=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/zboard-agent/agent.env
ExecStart=/usr/local/bin/zboard-agent --config /etc/zboard-agent/agent.env
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now zboard-agent
```

### 节点运行时

Agent 需要 Xray 或 sing-box 已安装：

```bash
# Xray
bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install

# sing-box
bash -c "$(curl -fsSL https://sing-box.app/deb-install.sh)"
```

## 添加节点流程

1. 管理员后台 `POST /api/admin/v1/nodes` 创建节点
2. 记录返回的 `node_id` + `node_secret`（仅返回一次）
3. 在 VPS 上部署 Agent，填入上述信息
4. Agent 自动注册、拉取配置、启动运行时
5. 管理员 `POST /api/admin/v1/nodes/:id/sync-config` 生成配置
6. Agent 下次 pull 时自动应用

## 环境变量参考

### API Server

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `ZBOARD_HOST` | 否 | `127.0.0.1` | 监听地址 |
| `ZBOARD_PORT` | 否 | `3000` | 监听端口 |
| `ZBOARD_DB_DIALECT` | 是 | `sqlite` | `mysql` / `postgres` / `sqlite` |
| `ZBOARD_DB_DSN` | 是* | - | 数据库连接串 |
| `ZBOARD_ADMIN_SETUP_TOKEN` | 是 | - | 首次初始化管理员密钥 |
| `ZBOARD_TOKEN_SECRET` | 是 | - | 会话 token 签名密钥 |

### Node Agent

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `ZBOARD_AGENT_API_BASE_URL` | 是 | - | 控制面地址 |
| `ZBOARD_AGENT_NODE_ID` | 是 | - | 节点 ID |
| `ZBOARD_AGENT_NODE_SECRET` | 是 | - | 节点密钥 |
| `ZBOARD_AGENT_RUNTIME_BINARY` | 否 | `/usr/local/bin/xray` | 运行时路径 |
| `ZBOARD_AGENT_RUNTIME_TYPE` | 否 | `xray` | `xray` 或 `sing-box` |
| `ZBOARD_AGENT_STATS_API_ADDR` | 否 | `127.0.0.1:10085` | stats gRPC 地址 |

## 数据库

线上推荐 MariaDB。启动时自动执行 migration，无需手动建表。

DSN 格式：`user:password@tcp(host:3306)/dbname?parseTime=true&charset=utf8mb4`

## 监控

- 健康检查：`GET /health`
- Agent 心跳：`nodes.last_heartbeat_at`
- 审计日志：`GET /api/admin/v1/audit-logs`

## CI/CD

GitHub Actions 在 push 到 `main` 时自动构建三个镜像并推送到 DockerHub。

生产更新：`docker compose pull && docker compose up -d`
