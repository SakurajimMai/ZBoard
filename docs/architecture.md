# 系统架构文档

## 总体架构

```text
用户前台
  ↓
用户中心 / 套餐购买 / 订阅查看
  ↓
商业 API Server
  ↓
订单系统 / 支付系统 / 套餐系统 / 用户系统
  ↓
订阅生成服务
  ↓
节点调度服务
  ↓
Agent Gateway
  ↓
自研 Node Agent
  ↓
Xray-core / sing-box core
  ↓
VPS 节点
```

## 控制面与数据面

### Control Plane 控制面

- `api-server`：商业主后端，承载用户、订单、支付、套餐、节点、订阅和后台 API。
- `admin-web`：管理后台。
- `user-web`：用户前台。
- `worker-service`：异步任务、到期检查、超额检查、配置生成任务。
- `agent-gateway`：Agent 注册、心跳、任务拉取、结果上报和流量上报入口。
- `sub-service`：订阅生成服务。
- `scheduler-service`：节点调度和周期性任务服务。

### Data Plane 数据面

- `node-agent`：部署在每台 VPS 上的自研 Agent。
- `runtime manager`：管理 Xray / sing-box 进程、配置和重载。
- `traffic collector`：采集用户维度流量增量。
- `config renderer`：在节点本地渲染或应用中央生成的运行时配置。
- `xray-core / sing-box core`：真正处理代理流量的运行时。

## MVP 服务边界

第一版不建议过早拆成大量微服务。推荐落地边界：

| 服务 | MVP 形态 | 后续演进 |
| --- | --- | --- |
| `api-server` | 独立服务 | 保持主业务入口 |
| `worker-service` | 独立服务 | 扩展更多队列和定时任务 |
| `admin-web` | 独立应用 | 保持独立部署 |
| `user-web` | 独立应用 | 保持独立部署 |
| `sub-service` | 先作为 `api-server` 模块 | 高并发后拆出 |
| `agent-gateway` | 先作为 `api-server` 模块 | Agent 规模扩大后拆出 |
| `scheduler-service` | 先作为 `worker-service` 模块 | 调度复杂后拆出 |
| `node-agent` | Go 单文件 + systemd | 保持节点侧独立 |

## 推荐技术栈

### 控制面

MVP 推荐：

- 后端：NestJS + TypeScript。
- ORM：Drizzle ORM 或 Prisma。
- 数据库：PostgreSQL。
- 缓存和锁：Redis。
- 队列：BullMQ。
- 前端：Next.js + React + Tailwind CSS + shadcn/ui。

说明：原始方案中的 Hono 适合轻量 API，但 Zboard 的商业模块、后台权限、支付回调、队列和审计复杂度较高，MVP 默认采用 NestJS 更利于模块化和团队协作。若后续需要边缘部署或极轻量网关，可以在局部服务中引入 Hono。

### 节点 Agent

- 语言：Go。
- 部署：Linux 单文件 + systemd。
- 配置：YAML。
- 通信：HTTPS + HMAC 签名。
- 运行时：Xray-core / sing-box core。

## 仓库结构

```text
zboard/
├── apps/
│   ├── user-web/                  # 用户前台
│   ├── admin-web/                 # 管理后台
│   ├── api-server/                # 商业主后端
│   ├── worker-service/            # 异步任务服务
│   └── node-agent/                # 自研节点 Agent
│
├── packages/
│   ├── shared-types/              # 公共类型
│   ├── sdk/                       # 内部 SDK
│   ├── config-builder/            # 节点配置生成器
│   ├── subscription-renderer/     # 订阅格式生成器
│   ├── payment-sdk/               # 支付适配器
│   ├── agent-protocol/            # Agent 通信协议
│   ├── runtime-schema/            # Xray / sing-box 配置模型
│   └── security-kit/              # 加密、签名、鉴权、限流工具
│
├── deploy/
│   ├── docker/
│   ├── systemd/
│   ├── k8s/
│   └── terraform/
│
├── docs/
└── scripts/
```

## 后端模块

```text
api-server/
├── src/
│   ├── modules/
│   │   ├── auth/                  # 登录注册
│   │   ├── users/                 # 用户管理
│   │   ├── plans/                 # 套餐管理
│   │   ├── orders/                # 订单管理
│   │   ├── payments/              # 支付系统
│   │   ├── subscriptions/         # 订阅 token
│   │   ├── nodes/                 # 节点管理
│   │   ├── node-groups/           # 节点组
│   │   ├── node-agents/           # Agent 管理
│   │   ├── node-tasks/            # 节点任务
│   │   ├── runtime-configs/       # 节点运行配置
│   │   ├── traffic/               # 流量统计
│   │   ├── admin/                 # 管理员
│   │   ├── audit-logs/            # 操作日志
│   │   └── system-settings/       # 系统配置
│   │
│   ├── services/
│   │   ├── config-builder/        # 配置生成
│   │   ├── node-dispatcher/       # 节点任务分发
│   │   ├── traffic-aggregator/    # 流量聚合
│   │   └── subscription-builder/  # 订阅生成
│   │
│   ├── common/
│   ├── database/
│   └── main.ts
```

## 核心流程

### 用户购买套餐

```text
用户选择套餐
  ↓
创建订单
  ↓
支付
  ↓
支付回调验签
  ↓
订单标记已支付
  ↓
开通套餐
  ↓
生成用户订阅 token
  ↓
创建用户节点权限
  ↓
生成节点同步任务
  ↓
Agent 拉取任务
  ↓
Agent 更新本机配置
  ↓
重载节点运行时
  ↓
订阅服务开始返回可用节点
```

### 节点配置同步

```text
后台新增节点
  ↓
生成 node_id / node_secret
  ↓
安装 Node Agent
  ↓
Agent 向 Agent Gateway 注册
  ↓
Agent 定时心跳
  ↓
中央系统下发任务
  ↓
Agent 拉取任务
  ↓
Agent 应用完整配置
  ↓
校验配置
  ↓
reload runtime
  ↓
上报执行结果
```

### 流量统计

```text
Node Agent 本地采集流量
  ↓
按用户维度生成增量数据
  ↓
上报 Agent Gateway
  ↓
写入 traffic_logs
  ↓
聚合到 user_traffic_snapshots
  ↓
判断是否超额
  ↓
超额后生成 disable_user_on_node 任务
  ↓
Agent 禁用用户
  ↓
订阅服务停止返回该用户节点
```

## 架构决策

- 第一版采用完整配置同步，不做细粒度 patch。
- 节点运行配置由中央系统生成 Xray / sing-box 最小完整配置，Agent 只负责校验、替换、reload 和回滚。
- Agent 主动拉取任务，中央系统不主动连接节点。
- 订阅生成可以先在主 API 中实现，流量变大后再拆服务。
- Worker 维护任务可以先在主 API 中实现，正式环境通过 cron 或独立 `worker-service` 调用同一服务函数。
- 节点运行时配置必须支持版本、hash、校验和回滚。
- 所有支付回调和节点任务必须幂等。
- 所有敏感后台操作必须进入审计日志。
