# Agent 协议文档

## 设计目标

Node Agent 部署在每台 VPS 上，负责和中央系统通信，并管理本机运行时。Agent 必须主动连接中央系统，中央系统不主动 SSH 到节点，也不要求节点暴露公网管理端口。

Agent 职责：

- 节点注册。
- 心跳上报。
- 拉取任务。
- 执行任务。
- 写入配置。
- 校验配置。
- 重载运行时。
- 采集流量。
- 上报流量。
- 进程守护。
- 失败回滚。
- 日志上报。

Agent 不做：

- 不处理订单。
- 不处理支付。
- 不保存用户密码。
- 不暴露管理后台。
- 不开放公网管理端口。

## 配置文件

```yaml
node_id: "node_abc123"
node_secret: "replace_with_secret"
gateway_url: "https://api.example.com/api/agent/v1"

runtime:
  type: "xray"
  binary_path: "/usr/local/bin/xray"
  config_path: "/etc/zboard/runtime/config.json"
  backup_dir: "/etc/zboard/runtime/backups"
  reload_command: "systemctl reload xray"

heartbeat:
  interval_seconds: 30

task:
  pull_interval_seconds: 10
  max_batch_size: 50

traffic:
  report_interval_seconds: 60

log:
  level: "info"
  path: "/var/log/zboard-agent/agent.log"
```

## 请求签名

Agent 所有请求都必须带签名。

请求头：

```http
X-Zboard-Node-Id: node_abc123
X-Zboard-Timestamp: 1777099200
X-Zboard-Nonce: random-128-bit-string
X-Zboard-Body-SHA256: hex-body-sha256
X-Zboard-Signature: hex-hmac-sha256
```

签名原文：

```text
METHOD
PATH_WITH_QUERY
TIMESTAMP
NONCE
BODY_SHA256
```

签名算法：

```text
HMAC-SHA256(node_secret_hash, canonical_string)
```

当前后端不会保存明文 `node_secret`，只保存 `sha256(node_secret)`。因此 Agent 侧签名时需要先计算 `node_secret_hash = sha256(node_secret)`，再使用该 hash 作为 HMAC 密钥。

服务端校验：

- `node_id` 必须存在且启用。
- timestamp 与服务端时间差不能超过 5 分钟。
- nonce 在有效窗口内不能重复。
- body hash 必须与请求体一致。
- HMAC 签名必须匹配。
- `node_id`、timestamp、nonce、body hash 和 signature 均从请求头读取。
- 请求体不再携带明文 `node_secret`。

## 生命周期

### 1. 节点创建

管理员在后台创建节点，系统生成：

- `node_id`
- `node_secret`
- 安装命令

`node_secret` 只展示一次，数据库只保存 hash。

### 2. Agent 注册

```http
POST /api/agent/v1/register
```

请求：

```json
{
  "agent_version": "0.1.0",
  "os_info": {
    "os": "linux",
    "arch": "amd64"
  },
  "runtime_info": {
    "type": "xray",
    "version": "1.8.0"
  }
}
```

返回：

```json
{
  "success": true,
  "data": {
    "server_time": "2026-04-25T12:00:00Z",
    "heartbeat_interval_seconds": 30,
    "task_pull_interval_seconds": 10
  }
}
```

### 3. 心跳

```http
POST /api/agent/v1/heartbeat
```

请求：

```json
{
  "agent_version": "0.1.0",
  "runtime_status": "running",
  "runtime_info": {
    "type": "xray",
    "version": "1.8.0",
    "config_hash": "sha256:xxx"
  },
  "system_load": {
    "cpu": 0.3,
    "memory": 0.5,
    "disk": 0.4
  },
  "reported_at": "2026-04-25T12:00:00Z"
}
```

### 4. 拉取任务

```http
POST /api/agent/v1/tasks/pull
```

请求：

```json
{
  "max_batch_size": 10
}
```

返回：

```json
{
  "success": true,
  "data": {
    "tasks": [
      {
        "task_id": "task_001",
        "task_type": "sync_full_config",
        "payload": {
          "version": "20260425120000",
          "config_hash": "sha256:xxx",
          "config": {}
        }
      }
    ]
  }
}
```

## 任务模型

MVP 任务类型：

| 任务类型 | 说明 |
| --- | --- |
| `sync_full_config` | 拉取并应用完整运行时配置 |
| `disable_user` | 在节点上禁用用户 |
| `delete_user` | 在节点上删除用户 |
| `reload_runtime` | 重载运行时 |

后续任务类型：

| 任务类型 | 说明 |
| --- | --- |
| `create_user` | 创建用户 |
| `enable_user` | 启用用户 |
| `update_user_limit` | 更新用户限速或设备限制 |
| `restart_runtime` | 重启运行时 |
| `upgrade_agent` | 升级 Agent |
| `rotate_node_secret` | 轮换节点密钥 |

## 任务执行要求

- 任务必须幂等。
- 同一个 `task_id` 重复执行时不得产生错误副作用。
- 配置替换前必须备份旧配置。
- 新配置必须先校验，再替换。
- 替换后 reload 失败必须回滚。
- 执行结果必须上报。
- 失败原因必须可读，便于后台排查。

## 完整配置同步

推荐 MVP 使用完整配置同步：

```text
中央系统生成完整配置
  ↓
Agent 拉取 sync_full_config 任务
  ↓
Agent 写入临时配置文件
  ↓
Agent 执行运行时配置校验
  ↓
Agent 备份当前配置
  ↓
Agent 替换配置
  ↓
Agent reload runtime
  ↓
Agent 上报执行结果
```

比增量 patch 更稳，因为中央系统始终是配置真源。

## 任务结果上报

```http
POST /api/agent/v1/tasks/{task_id}/result
```

成功：

```json
{
  "node_id": "node_abc123",
  "status": "success",
  "message": "配置已应用",
  "detail": {
    "version": "20260425120000",
    "config_hash": "sha256:xxx"
  },
  "reported_at": "2026-04-25T12:00:00Z"
}
```

失败：

```json
{
  "node_id": "node_abc123",
  "status": "failed",
  "message": "运行时配置校验失败",
  "detail": {
    "stderr": "invalid config"
  },
  "reported_at": "2026-04-25T12:00:00Z"
}
```

## 流量上报

```http
POST /api/agent/v1/traffic/report
```

请求：

```json
{
  "node_id": "node_abc123",
  "records": [
    {
      "client_id": "uuid",
      "upload_delta": 1024,
      "download_delta": 2048,
      "reported_at": "2026-04-25T12:00:00Z"
    }
  ]
}
```

处理要求：

- 服务端用 `client_id` 映射 `user_id` 和 `node_id`。
- 增量必须大于等于 0。
- 同一批次上报需要幂等键或批次 ID。
- 聚合失败不能丢原始日志。

## 失败和重试

- 网络失败：Agent 指数退避重试。
- 任务执行失败：上报失败结果，由服务端决定是否重试。
- 配置校验失败：不替换当前配置。
- reload 失败：回滚旧配置并上报失败。
- 服务端返回 401 / 403：Agent 停止执行任务，只保留心跳和错误日志。

## 密钥轮换

密钥轮换应作为后续功能实现：

1. 后台生成新密钥。
2. 下发 `rotate_node_secret` 任务。
3. Agent 使用旧密钥签名领取任务。
4. Agent 保存新密钥。
5. Agent 使用新密钥发送确认请求。
6. 服务端启用新密钥并废弃旧密钥。
