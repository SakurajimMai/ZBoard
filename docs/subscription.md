# 订阅系统文档

## 目标

订阅系统根据用户套餐、节点权限、客户端类型和节点状态生成不同格式的订阅配置。订阅服务是商业系统面向用户的核心出口，必须保证安全、稳定、可缓存、可审计。

## 支持格式

MVP 优先支持：

- Clash Meta。
- sing-box。
- 通用 Base64。

后续支持：

- V2RayN。
- Shadowrocket。
- Stash。
- Surge。
- Quantumult X。

## 路由

```http
GET /api/sub/{token}
GET /api/sub/{token}?target=clash
GET /api/sub/{token}?target=singbox
GET /api/sub/{token}?target=v2ray
```

当未传 `target` 时，服务端可以根据 `User-Agent` 自动识别客户端；识别失败时返回默认格式，MVP 默认可使用 Clash Meta 或通用 Base64。

## 生成流程

```text
校验 token
  ↓
读取用户状态
  ↓
检查套餐是否过期
  ↓
检查流量是否超额
  ↓
读取用户可访问节点组
  ↓
过滤维护中节点
  ↓
过滤 Agent 离线节点
  ↓
读取节点连接参数
  ↓
根据 target 生成配置
  ↓
记录订阅访问日志
  ↓
返回订阅内容
```

## 用户可用性判断

用户满足以下条件才返回可用节点：

- 用户状态为 `active`。
- 套餐未过期。
- 流量未超额。
- 订阅 token 状态为 `active`。
- 用户所属套餐绑定的节点组存在可用节点。

过期或超额用户可以返回空配置，也可以返回提示配置。MVP 推荐返回空配置并记录原因，避免客户端行为差异导致误解。

## 节点可用性判断

节点满足以下条件才进入订阅：

- 节点状态为 `active`。
- 节点不处于维护模式。
- Agent 最近心跳未超时。
- 节点运行时状态正常。
- 用户套餐允许访问该节点组。

## 生成器接口

```ts
export interface SubscriptionRenderInput {
  user: {
    id: number;
    email: string;
    expiredAt: Date | null;
    trafficLimit: bigint;
    trafficUsed: bigint;
  };
  nodes: SubscriptionNode[];
  target: SubscriptionTarget;
  userAgent?: string;
}

export type SubscriptionTarget =
  | 'clash'
  | 'singbox'
  | 'v2ray'
  | 'base64';

export interface SubscriptionRenderer {
  render(input: SubscriptionRenderInput): Promise<SubscriptionRenderResult>;
}

export interface SubscriptionRenderResult {
  contentType: string;
  body: string;
  headers?: Record<string, string>;
}
```

## 节点输出模型

```ts
export interface SubscriptionNode {
  id: number;
  name: string;
  region?: string;
  host: string;
  port: number;
  protocol: 'vless' | 'vmess' | 'trojan' | 'shadowsocks';
  transport: 'tcp' | 'ws' | 'grpc';
  security: 'none' | 'tls' | 'reality';
  clientId: string;
  password?: string;
  flow?: string;
  sni?: string;
  publicKey?: string;
  shortId?: string;
  path?: string;
  serviceName?: string;
}
```

## Clash Meta 输出

返回：

```http
Content-Type: text/yaml; charset=utf-8
```

要求：

- 节点名称应包含地区和节点名。
- 同名节点需要自动去重。
- 需要输出基础代理组。
- 维护节点不输出。
- 超额或过期用户返回空 proxies。
- 当前 API Server 已输出 `type`、`server`、`port`、`uuid`、`encryption`、`network`、`tls` 等基础字段。

## sing-box 输出

返回：

```http
Content-Type: application/json; charset=utf-8
```

要求：

- 输出合法 JSON。
- 节点协议字段必须与 sing-box schema 对齐。
- 对 Reality、TLS、WebSocket、gRPC 做明确映射。
- 当前 API Server 已输出 `type`、`tag`、`server`、`server_port`、`uuid`、`tls` 和基础 `transport`。

## Base64 输出

返回：

```http
Content-Type: text/plain; charset=utf-8
```

要求：

- 每行一个节点分享链接。
- 整体 Base64 编码。
- 节点顺序稳定。
- 当前 API Server 已输出 VLESS 分享链接，包含 `encryption=none`、`security`、`type` 和节点显示名。

## 缓存策略

MVP 推荐：

- 有效订阅缓存 30-120 秒。
- 用户套餐、流量、节点状态变化后主动删除缓存。
- token 重置后立即删除旧 token 缓存。
- 缓存 key 包含 token hash、target、客户端识别结果。

缓存 key 示例：

```text
sub:{token_hash}:{target}:{ua_family}
```

## 限流策略

建议按以下维度限流：

- token。
- IP。
- token + IP。

MVP 可以先实现：

- 单 token 每分钟 30 次。
- 单 IP 每分钟 120 次。
- 异常访问写入订阅访问日志。

## 访问日志

每次请求记录：

- 用户 ID。
- token hash。
- target。
- IP。
- User-Agent。
- 结果。
- 失败原因。
- 创建时间。

## token 策略

- token 使用高强度随机值。
- 数据库只保存 token hash。
- 用户可主动重置 token。
- 重置后旧 token 立即失效。
- 管理后台可禁用用户 token。

## 异常返回策略

| 场景 | 推荐返回 |
| --- | --- |
| token 不存在 | 404 或空配置 |
| token 被禁用 | 空配置 |
| 用户被禁用 | 空配置 |
| 套餐过期 | 空配置 |
| 流量超额 | 空配置 |
| 节点全部不可用 | 空配置 |
| 系统错误 | 500，记录错误日志 |

为了减少客户端暴露的信息，订阅接口不应向用户返回过多内部原因。详细原因写入后台日志。
