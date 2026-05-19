# 部署方案文档

## 部署原则

- 本地开发不执行 Docker，直接使用 Node.js + SQLite3。
- 正式环境控制面使用 Docker Compose。
- 当前阶段不配置反向代理，不引入 Nginx / Caddy / Traefik。
- Node Agent 生产环境优先 systemd 部署。
- 数据库迁移和初始化管理员需要可重复执行。
- 日志、备份、健康检查必须有明确位置。

## 当前 Docker 化边界

当前 Compose 只包含：

```text
api-server
```

当前不包含：

```text
nginx
caddy
traefik
postgres
redis
worker-service
admin-web
user-web
```

说明：

- 第一版后端仍使用 SQLite3 文件，数据库文件挂载到 `/data/zboard.sqlite`。
- Worker 维护任务当前在 API Server 内部，通过后台接口手动触发。
- PostgreSQL、Redis、独立 Worker 和 Web 前端等正式拆分后再加入 Compose。

## 目录结构

```text
deploy/
├── docker/
│   └── docker-compose.prod.yml
├── env/
│   └── api.env.example
└── scripts/
    └── backup-sqlite.sh
```

## 环境变量

复制示例文件：

```bash
cp deploy/env/api.env.example deploy/env/api.env
```

必须修改：

```env
ZBOARD_ADMIN_SETUP_TOKEN=replace-with-one-time-setup-token
ZBOARD_TOKEN_SECRET=replace-with-random-token-secret
```

当前 API Server 使用：

```env
ZBOARD_HOST=0.0.0.0
ZBOARD_PORT=3000
ZBOARD_DB_PATH=/data/zboard.sqlite
ZBOARD_BACKUP_DIR=/backups
```

## 生产启动

生产环境启动命令：

```bash
docker compose -f deploy/docker/docker-compose.prod.yml up -d --build
```

查看状态：

```bash
docker compose -f deploy/docker/docker-compose.prod.yml ps
```

查看日志：

```bash
docker compose -f deploy/docker/docker-compose.prod.yml logs -f api-server
```

## 初始化管理员

服务启动后调用：

```http
POST /api/admin/v1/auth/bootstrap
```

请求体需要提供 `ZBOARD_ADMIN_SETUP_TOKEN` 对应的一次性初始化令牌。系统中已有管理员后，该接口返回 409。

## 数据库迁移

当前 SQLite migration 路径：

```text
apps/api-server/src/db/migrations/sqlite
```

API Server 启动时自动应用未执行 migration，并记录到 `schema_migrations`。

PostgreSQL migration 预留路径：

```text
apps/api-server/src/db/migrations/postgres
```

正式环境切 PostgreSQL 前，需要先补齐同版本 PostgreSQL SQL，并在预发布数据库执行完整迁移和回归测试。

## 健康检查

API Server 健康检查：

```http
GET /health
```

Compose 已配置容器 healthcheck：

```text
http://127.0.0.1:3000/health
```

健康检查用于判断 API 进程是否可响应；数据库连通性已经在应用启动和业务请求中覆盖，后续可以把数据库状态加入 `/health` 返回。

## 日志

Compose 使用 Docker `json-file` 日志驱动，并配置轮转：

```text
max-size=10m
max-file=5
```

查看日志：

```bash
docker compose -f deploy/docker/docker-compose.prod.yml logs --tail=200 api-server
```

## 备份

必须备份：

- `/data/zboard.sqlite`
- `deploy/env/api.env` 的安全副本

备份脚本：

```text
deploy/scripts/backup-sqlite.sh
```

容器内执行示例：

```bash
docker compose -f deploy/docker/docker-compose.prod.yml exec api-server sh deploy/scripts/backup-sqlite.sh
```

脚本行为：

- 从 `ZBOARD_DB_PATH` 读取 SQLite 文件。
- 输出到 `ZBOARD_BACKUP_DIR`。
- 使用 gzip 压缩。
- 删除 30 天前的旧备份。

也可以在宿主机直接备份 Docker volume，生产环境应至少保留 7-30 天。

## Agent systemd 部署

服务文件目标：

```ini
[Unit]
Description=Zboard Node Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/zboard-agent -config /etc/zboard/agent.yaml
Restart=always
RestartSec=5
User=root
WorkingDirectory=/var/lib/zboard-agent

[Install]
WantedBy=multi-user.target
```

Agent 需要管理宿主机上的运行时、写配置、调用 systemd、采集本机流量。第一版生产环境不建议把 Agent 放进 Docker。

## 上线检查表

- `deploy/env/api.env` 已创建且未提交仓库。
- `ZBOARD_ADMIN_SETUP_TOKEN` 已设置为一次性强随机值。
- `ZBOARD_TOKEN_SECRET` 已设置为强随机值。
- Docker Compose 已启动。
- `/health` 返回正常。
- 数据库迁移已自动应用。
- 管理员账号已创建。
- 备份脚本可执行。
- 日志轮转已生效。
