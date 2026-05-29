# Zboard Node Agent

Pure Go agent that runs on each VPS. It connects **out** to the control plane
(no inbound SSH from Zboard), reports heartbeats, pulls config-sync tasks, and
supervises the underlying Xray / sing-box runtime as a child process.

## Build

```bash
cd apps/node-agent
go build -o zboard-agent ./cmd/zboard-agent
```

The binary is self-contained — no CGO, no external dependencies.

## Configure

Copy `deploy/agent.env.example` to `/etc/zboard-agent/agent.env` and fill in:

```
ZBOARD_AGENT_API_BASE_URL=https://control.example.com
ZBOARD_AGENT_NODE_ID=<id from POST /api/admin/v1/nodes>
ZBOARD_AGENT_NODE_SECRET=<plaintext returned ONCE on create>
ZBOARD_AGENT_RUNTIME_BINARY=/usr/local/bin/xray   # or /usr/local/bin/sing-box
ZBOARD_AGENT_RUNTIME_TYPE=xray                    # or sing-box
```

`NODE_SECRET` is the plaintext value the control plane returns at node-create
time. It is **not** stored on the server (only `sha256(secret)` is).

## Install (systemd)

```bash
install -Dm755 zboard-agent /usr/local/bin/zboard-agent
install -Dm644 deploy/zboard-agent.service /etc/systemd/system/zboard-agent.service
useradd --system --no-create-home --shell /usr/sbin/nologin zboard || true
install -d -o zboard -g zboard /etc/zboard-agent /var/lib/zboard-agent
install -m640 -o zboard -g zboard deploy/agent.env.example /etc/zboard-agent/agent.env
systemctl daemon-reload
systemctl enable --now zboard-agent
journalctl -u zboard-agent -f
```

## Wire protocol

All `/api/agent/v1/*` writes are signed via HMAC-SHA256 over
`${node_id}|${ts}|${nonce}|${body_sha256}|POST|${path}` with key
`hex(sha256(node_secret))`. Headers:

```
X-Zboard-Node-Id
X-Zboard-Timestamp        (5-minute window)
X-Zboard-Nonce            (server-side dedup via agent_nonces)
X-Zboard-Body-SHA256
X-Zboard-Signature
```

The agent's three loops:

| Loop      | Interval        | Endpoint                         |
| --------- | --------------- | -------------------------------- |
| Heartbeat | `30s` default   | `POST /api/agent/v1/heartbeat`   |
| Tasks     | `10s` default   | `POST /api/agent/v1/tasks/pull`  |
| Traffic   | `60s` default   | `POST /api/agent/v1/traffic/report` |

`sync_config` tasks include the full runtime config inline; the supervisor
hashes it and skips the restart when nothing changed. `disable_user` tasks are
acknowledged immediately because the server already excluded the disabled user
from the next config push.

## Limitations

- Runtime restart is full-process, not graceful drain. 1-2s gap per config
  swap; acceptable for MVP, revisit if SLO needs tighter.
- TLS verification uses the system trust store; pin / mTLS to the control
  plane is a follow-up.

## Per-user traffic accounting

The control plane's `internal/runtime` package builds Xray and sing-box configs
that expose a local stats gRPC API on `127.0.0.1:10085`:

- **Xray**: `stats: {}` + `api: {tag, services: [StatsService]}` +
  `policy.levels.0.statsUserUplink/Downlink: true` + a `dokodemo-door`
  inbound tagged `api` that the `routing.rules` block routes to the api
  outbound. Each client carries `email: u<user_id>@zboard`, so Xray emits
  stats named `user>>>u<id>@zboard>>>traffic>>>{uplink,downlink}`.
- **sing-box**: the bundled Docker image builds sing-box with
  `with_v2ray_api` and sets `ZBOARD_AGENT_SINGBOX_V2RAY_API=1`, so generated
  configs keep `experimental.v2ray_api.stats.users` using names like
  `u<user_id>`. Hand-installed agents default this capability off; set the env
  flag only for a sing-box binary compiled with `with_v2ray_api`, otherwise the
  agent strips the unsupported config block before starting the runtime.

Each `traffic_report` cycle the agent calls the runtime-specific
`StatsService/QueryStats(pattern="user>>>", reset=true)` method over gRPC
(Xray's service path for Xray, v2ray's service path for sing-box), parses
each `user>>>u<id>...>>>traffic>>>{uplink|downlink}` row, groups by
`user_id`, and POSTs the deltas to `/api/agent/v1/traffic/report`. The
`reset=true` flag means counters return to zero on every read, so the agent
never double-counts even across runtime restarts.

The protobuf wire format is hand-coded with
`google.golang.org/protobuf/encoding/protowire` to avoid pulling in
`xray-core`; the unit + integration tests in `internal/stats` cover both
encode and decode against an in-process gRPC server.
