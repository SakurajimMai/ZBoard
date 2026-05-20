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
- `frontend`：Next.js 用户端和管理端，按 `/dashboard`、`/admin` 等路由区分业务入口。
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
| `frontend` | 独立 Next.js 应用 | 后续可按访问量拆分用户端和管理端 |
| `sub-service` | 先作为 `api-server` 模块 | 高并发后拆出 |
| `agent-gateway` | 先作为 `api-server` 模块 | Agent 规模扩大后拆出 |
| `scheduler-service` | 先作为 `worker-service` 模块 | 调度复杂后拆出 |
| `node-agent` | Go 单文件 + systemd | 保持节点侧独立 |

## 推荐技术栈

### 控制面

当前实现：

- 后端：Go 1.25 + Gin + sqlx。
- 数据库：MySQL / MariaDB / PostgreSQL / SQLite，按方言维护 migration。
- 任务：维护任务运行在 API Server 内，负责到期/超额停用、任务超时清理和失败重试。
- 前端：Next.js 16 + React 19 + Tailwind 4 + shadcn/ui。
- 部署：API Server、前端和 Node Agent 都提供 Dockerfile，正式环境通过 Docker Compose 或 systemd 运行。

说明：当前代码已经落地为 Go 控制面和独立 Next.js 前端，不再使用早期草案中的 NestJS/Hono 方案。后续如需独立 worker、边缘网关或更多拆分服务，应在保持 API 契约稳定的前提下增量演进。

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
│   ├── api-server/                # Go 控制面 API
│   └── node-agent/                # Go 节点 Agent
│
├── frontend/                      # Next.js 用户端和管理端
│
├── deploy/
│   └── docker/
│
├── docs/
└── scripts/
```

## 后端模块

```text
api-server/
├── cmd/server/                    # 启动入口
└── internal/
    ├── authsvc/                   # 用户/管理员登录注册
    ├── bizsvc/                    # 套餐、订单、支付激活
    ├── db/                        # sqlx 连接和三方言 migration
    ├── nodesvc/                   # 配置同步和回滚任务
    ├── runtime/                   # Xray / sing-box 配置生成
    ├── subrender/                 # Clash / sing-box / Base64 订阅渲染
    ├── agentauth/                 # Agent HMAC 鉴权
    ├── store/                     # 仓储层
    ├── worker/                    # 到期/超额/任务维护
    └── server/                    # Gin 路由和 HTTP handler
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
