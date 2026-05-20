# API 设计文档

## 设计约定

- API 使用 JSON。
- 时间使用 ISO 8601 字符串。
- 金额字段使用字符串或定点数，避免浮点误差。
- 流量字段单位为字节。
- 用户端接口前缀：`/api/v1`。
- 管理后台接口前缀：`/api/admin/v1`。
- Agent 接口前缀：`/api/agent/v1`。
- 订阅接口前缀：`/api/sub`。

## 前端入口

当前前端是独立 Next.js 工程，位于 `frontend/`：

```text
/                  # 落地页
/login             # 用户登录
/dashboard         # 用户控制台
/admin/login       # 管理员登录
/admin             # 管理后台
```

前端通过 `NEXT_PUBLIC_API_URL` 指向控制面 API。该变量会暴露给浏览器，正式环境必须填写用户浏览器可访问的地址；留空时前端会按当前主机推导 `:3000` 端口作为 API 地址。

## 通用响应

成功：

```json
{
  "success": true,
  "data": {}
}
```

失败：

```json
{
  "success": false,
  "error": {
    "code": "ORDER_NOT_FOUND",
    "message": "订单不存在"
  }
}
```

## 通用错误码

| 错误码 | 说明 |
| --- | --- |
| `UNAUTHORIZED` | 未登录或凭证无效 |
| `FORBIDDEN` | 权限不足 |
| `VALIDATION_ERROR` | 请求参数错误 |
| `RESOURCE_NOT_FOUND` | 资源不存在 |
| `RATE_LIMITED` | 请求过于频繁 |
| `IDEMPOTENCY_CONFLICT` | 幂等键冲突 |
| `INTERNAL_ERROR` | 服务内部错误 |

## 用户端 API

### 注册

```http
POST /api/v1/auth/register
```

请求：

```json
{
  "email": "user@example.com",
  "password": "password"
}
```

### 登录

```http
POST /api/v1/auth/login
```

请求：

```json
{
  "email": "user@example.com",
  "password": "password"
}
```

### 当前用户

```http
GET /api/v1/me
```

返回用户基础信息、套餐、到期时间、流量限额和已用流量。

### 套餐列表

```http
GET /api/v1/plans
```

只返回 `active` 状态套餐。

### 创建订单

```http
POST /api/v1/orders
Idempotency-Key: user-generated-key
```

请求：

```json
{
  "plan_id": 1,
  "payment_provider": "alipay"
}
```

### 订单详情

```http
GET /api/v1/orders/{order_no}
```

### 发起支付

```http
POST /api/v1/orders/{order_no}/pay
Idempotency-Key: user-generated-key
```

返回支付跳转链接、二维码内容或支付表单参数。

当前本地测试版本返回 mock 支付信息，用于验证支付前置记录和幂等行为。

### 用户订阅信息

```http
GET /api/v1/subscription
```

返回订阅链接、token 状态、最近访问 IP、最近访问 User-Agent。

### 重置订阅 token

```http
POST /api/v1/subscription/reset-token
Idempotency-Key: user-generated-key
```

重置后旧 token 立即失效。

## 管理后台 API

后台接口使用管理员 Bearer token，不再使用静态 `x-admin-token`。

### 初始化管理员

```http
POST /api/admin/v1/auth/bootstrap
```

请求：

```json
{
  "email": "admin@example.com",
  "password": "admin-passw0rd",
  "setup_token": "replace-with-setup-token"
}
```

只有系统中还没有管理员时允许初始化。初始化成功后会创建 `owner` 管理员，并返回后台会话 token。

### 管理员登录

```http
POST /api/admin/v1/auth/login
```

请求：

```json
{
  "email": "admin@example.com",
  "password": "admin-passw0rd"
}
```

### 当前管理员

```http
GET /api/admin/v1/auth/me
Authorization: Bearer <admin-token>
```

### 数据总览

```http
GET /api/admin/v1/dashboard
```

返回用户数、订单数、收入、节点数、在线 Agent、今日流量、失败任务数。

### 用户管理

```http
GET /api/admin/v1/users
GET /api/admin/v1/users/{id}
PATCH /api/admin/v1/users/{id}
POST /api/admin/v1/users/{id}/disable
POST /api/admin/v1/users/{id}/enable
```

所有写操作必须记录 `audit_logs`。

### 套餐管理

```http
GET /api/admin/v1/plans
POST /api/admin/v1/plans
PATCH /api/admin/v1/plans/{id}
POST /api/admin/v1/plans/{id}/disable
POST /api/admin/v1/plans/{id}/enable
```

### 订单管理

```http
GET /api/admin/v1/orders
GET /api/admin/v1/orders/{order_no}
```

MVP 不提供后台直接改订单为已支付，避免绕过支付审计。

### 支付记录

```http
GET /api/admin/v1/payments
GET /api/admin/v1/payment-callbacks
```

支付回调会先记录原始 callback，再执行订单校验和开通逻辑。重复 callback 根据 `provider_event_id` 幂等处理。

### 节点管理

```http
GET /api/admin/v1/nodes
POST /api/admin/v1/nodes
GET /api/admin/v1/nodes/{id}
PATCH /api/admin/v1/nodes/{id}
POST /api/admin/v1/nodes/{id}/maintenance
POST /api/admin/v1/nodes/{id}/enable
POST /api/admin/v1/nodes/{id}/disable
```

### 生成 Agent 安装信息

```http
POST /api/admin/v1/nodes/{id}/agent/bootstrap
```

返回一次性的 `node_id`、`node_secret` 和安装命令。`node_secret` 只展示一次。

### 重新同步节点配置

```http
POST /api/admin/v1/nodes/{id}/sync-config
Idempotency-Key: admin-generated-key
```

生成 `sync_full_config` 节点任务。

### 节点任务

```http
GET /api/admin/v1/node-tasks
GET /api/admin/v1/node-tasks/{task_id}
POST /api/admin/v1/node-tasks/{task_id}/retry
```

### 运行配置版本

```http
GET /api/admin/v1/nodes/{id}/runtime-configs
GET /api/admin/v1/runtime-configs/{version}
POST /api/admin/v1/runtime-configs/{version}/rollback
```

当前已支持按节点查询配置版本，以及按版本生成回滚同步任务。回滚不会直接修改节点运行时，而是生成新的 `sync_full_config` 任务，由 Agent 拉取后应用。

### 流量统计

```http
GET /api/admin/v1/traffic/users
GET /api/admin/v1/traffic/nodes
GET /api/admin/v1/traffic/logs
```

### 操作日志

```http
GET /api/admin/v1/audit-logs
```

### Worker 维护任务

```http
POST /api/admin/v1/workers/maintenance/run
```

当前 API Server 内部已提供可调用维护任务，用于：

- 禁用已到期用户。
- 禁用流量超额用户。
- 为被禁用用户生成节点 `disable_user` 任务。
- 将未达最大重试次数的失败任务重新入队。
- 将超时运行中的任务标记为失败。

## Agent API

Agent API 使用 HMAC 签名，具体规则见 [Agent 协议](./agent-protocol.md)。

Agent 请求必须通过以下请求头认证，不接受请求体明文 `node_secret`：

```http
X-Zboard-Node-Id
X-Zboard-Timestamp
X-Zboard-Nonce
X-Zboard-Body-SHA256
X-Zboard-Signature
```

### Agent 注册

```http
POST /api/agent/v1/register
```

请求：

```json
{
  "agent_version": "0.1.0",
  "os_info": {},
  "runtime_info": {}
}
```

### 心跳

```http
POST /api/agent/v1/heartbeat
```

请求：

```json
{
  "agent_version": "0.1.0",
  "runtime_status": "running",
  "runtime_info": {},
  "system_load": {},
  "reported_at": "2026-04-25T12:00:00Z"
}
```

### 拉取任务

```http
POST /api/agent/v1/tasks/pull
```

请求：

```json
{
  "max_batch_size": 10
}
```

### 上报任务结果

```http
POST /api/agent/v1/tasks/{task_id}/result
```

请求：

```json
{
  "node_id": "node_abc123",
  "status": "success",
  "message": "配置已应用",
  "detail": {
    "config_hash": "sha256:xxx"
  },
  "reported_at": "2026-04-25T12:00:00Z"
}
```

### 上报流量

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

## 订阅 API

### 获取订阅

```http
GET /api/sub/{token}
GET /api/sub/{token}?target=clash
GET /api/sub/{token}?target=singbox
GET /api/sub/{token}?target=v2ray
```

订阅接口需要记录：

- token hash。
- 用户 ID。
- IP。
- User-Agent。
- target。
- 返回结果。
- 失败原因。

## 支付回调 API

### 支付渠道回调

```http
POST /api/v1/payments/{provider}/callback
```

处理要求：

- 先保存原始回调。
- 再验签。
- 再查订单。
- 校验金额。
- 校验订单状态。
- 幂等更新订单和支付记录。
- 开通套餐。
- 生成订阅 token。
- 生成节点配置同步任务。

## 幂等策略

以下接口必须支持幂等：

- 创建订单。
- 发起支付。
- 重置订阅 token。
- 后台重新同步节点配置。
- 支付回调。
- Agent 上报任务结果。
- Agent 上报流量。

用户请求使用 `Idempotency-Key`。外部回调使用支付渠道事件 ID。Agent 请求使用 `node_id + nonce + timestamp + body_hash` 防重放。
