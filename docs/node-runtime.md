# 节点运行时文档

## 目标

节点运行时层负责把中央系统生成的用户和节点配置应用到本机 Xray-core 或 sing-box core。运行时层由 Node Agent 管理，必须支持配置校验、版本记录、原子替换、重载、失败回滚和流量采集。

## Agent 项目结构

```text
node-agent/
├── cmd/
│   └── agent/
│       └── main.go
│
├── internal/
│   ├── config/              # Agent 配置
│   ├── auth/                # 签名鉴权
│   ├── heartbeat/           # 心跳
│   ├── task/                # 任务拉取与执行
│   ├── runtime/             # Xray / sing-box 管理
│   ├── renderer/            # 本地配置渲染
│   ├── traffic/             # 流量采集
│   ├── process/             # 进程守护
│   ├── rollback/            # 配置回滚
│   ├── logger/              # 日志
│   └── updater/             # Agent 升级
│
├── configs/
│   └── agent.example.yaml
│
├── scripts/
│   ├── install.sh
│   ├── uninstall.sh
│   └── upgrade.sh
│
└── systemd/
    └── zboard-agent.service
```

## 配置生成器

中央系统需要提供 `config-builder`：

```ts
interface RuntimeConfigBuilder {
  buildNodeConfig(input: BuildNodeConfigInput): Promise<RuntimeConfigResult>;
  validateConfig(config: unknown): Promise<ValidateResult>;
  calculateHash(config: unknown): string;
}

interface BuildNodeConfigInput {
  node: NodeInfo;
  users: NodeUserInfo[];
  protocol: 'vless' | 'vmess' | 'trojan' | 'shadowsocks';
  runtimeType: 'xray' | 'singbox';
}

interface RuntimeConfigResult {
  version: string;
  hash: string;
  config: Record<string, unknown>;
}
```

MVP 推荐由中央系统生成完整配置，Agent 只负责校验和应用。这样可以减少节点侧逻辑复杂度。

当前后端已实现最小配置生成器：

- Xray：生成 `log`、单个 `inbounds`、基础 `outbounds`。
- sing-box：生成 `log`、单个 `inbounds`、基础 `outbounds`。
- 配置 hash 使用完整 JSON 内容计算。
- 每次同步生成 `runtime_configs` 版本记录。
- 回滚通过后台生成新的 `sync_full_config` 任务，由 Agent 应用。

## 完整配置应用流程

```text
Agent 收到 sync_full_config 任务
  ↓
检查 task_id 是否已执行
  ↓
写入临时配置文件
  ↓
执行 runtime validate
  ↓
备份当前配置
  ↓
原子替换配置文件
  ↓
执行 reload_command
  ↓
检查运行时状态
  ↓
成功则记录版本
  ↓
失败则回滚旧配置
  ↓
上报任务结果
```

后台接口：

```http
GET /api/admin/v1/nodes/{id}/runtime-configs
POST /api/admin/v1/runtime-configs/{version}/rollback
```

## 配置文件路径

推荐路径：

```text
/etc/zboard/agent.yaml
/etc/zboard/runtime/config.json
/etc/zboard/runtime/backups/
/var/log/zboard-agent/agent.log
/var/lib/zboard-agent/state.db
```

`state.db` 可使用 SQLite 或本地 JSON 状态文件，记录：

- 已执行任务 ID。
- 当前配置版本。
- 当前配置 hash。
- 最近成功上报的流量游标。

## Xray 校验

Xray 配置应用前必须校验：

```bash
xray test -config /path/to/config.json
```

如果运行时版本使用不同参数，需要在 Agent 中按版本适配。

## sing-box 校验

sing-box 配置应用前必须校验：

```bash
sing-box check -c /path/to/config.json
```

## 重载策略

MVP 推荐使用 systemd：

```text
systemctl reload xray
systemctl restart xray
systemctl reload sing-box
systemctl restart sing-box
```

如果运行时不支持 reload，则降级为 restart。降级行为必须记录到任务结果。

## 回滚策略

回滚触发场景：

- 新配置校验失败。
- 替换配置失败。
- reload 失败。
- reload 后运行时健康检查失败。

回滚动作：

1. 恢复上一份成功配置。
2. 执行 reload 或 restart。
3. 上报失败任务结果。
4. 记录错误日志和 stderr。

## 流量采集

候选方案：

| 方案 | 优点 | 缺点 | MVP 建议 |
| --- | --- | --- | --- |
| Xray stats API | 用户维度清晰 | 需要开启 stats 和 API | 推荐优先验证 |
| sing-box API | 与 sing-box 集成自然 | 版本差异需要确认 | sing-box 节点使用 |
| 日志解析 | 不依赖 API | 延迟和准确性较差 | 不推荐作为主方案 |
| nftables / iptables | 可统计链路流量 | 用户维度映射复杂 | 后续增强 |

MVP 建议：

- Xray 节点优先走 stats API。
- sing-box 节点优先走 sing-box 自身指标能力。
- Agent 上报用户维度增量，服务端负责聚合。

## 流量上报要求

- 上报增量，不直接覆盖总量。
- 增量不能为负数。
- Agent 本地需要保存上次采集游标。
- 运行时重启后需要处理计数器归零。
- 服务端聚合时必须保证幂等。

## 运行时状态

Agent 心跳需要上报：

- 运行时类型。
- 运行时版本。
- 运行时进程状态。
- 当前配置 hash。
- 当前配置版本。
- 最近 reload 时间。
- 最近错误信息。

## 进程守护

MVP 中进程守护由 systemd 承担。Agent 只负责检查运行时状态和上报，不直接替代 systemd。

后续可以增强：

- 自动拉起运行时。
- 异常崩溃告警。
- 连续失败后进入维护状态。

## 生产部署建议

生产环境不建议优先使用 Docker 部署 Agent，因为 Agent 需要管理宿主机运行时、写配置、调用 systemd 和采集本机流量。第一版更适合：

```text
Go 单文件
+ systemd
+ 宿主机配置目录
+ 最小权限运行
```

Docker Agent 可以作为开发和测试方案，但不作为生产默认部署方式。
