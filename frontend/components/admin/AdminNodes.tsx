"use client"

import { useEffect, useMemo, useState } from "react"
import type { ReactNode } from "react"
import { Copy, Edit3, Plus, RefreshCw, Server, Lock, Network, Sliders, Eye, EyeOff, Wand2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { AdminPager } from "@/components/admin/AdminPager"
import { adminCreateNode, adminGenerateRealityConfig, adminGetNodes, adminSyncAllNodeConfigs, adminSyncNodeConfig, adminUpdateNode } from "@/lib/api"

type NodeForm = {
  name: string
  region: string
  host: string
  port: string
  protocol: string
  transport: string
  security: string
  runtime_type: string
  status: "active" | "inactive"
  ws_path: string
  ws_host: string
  grpc_service_name: string
  sni: string
  fingerprint: string
  reality_public_key: string
  reality_short_id: string
  reality_server_name: string
  reality_private_key: string
  reality_dest: string
  flow: string
  alpn: string
  mux_enabled: string
  ss_method: string
  obfs_password: string
  congestion_control: string
  up_mbps: string
  down_mbps: string
  port_range: string
}

const emptyForm: NodeForm = {
  name: "",
  region: "",
  host: "",
  port: "443",
  protocol: "vless",
  transport: "tcp",
  security: "tls",
  runtime_type: "xray",
  status: "active",
  ws_path: "/",
  ws_host: "",
  grpc_service_name: "",
  sni: "",
  fingerprint: "chrome",
  reality_public_key: "",
  reality_short_id: "",
  reality_server_name: "",
  reality_private_key: "",
  reality_dest: "",
  flow: "",
  alpn: "",
  mux_enabled: "0",
  ss_method: "",
  obfs_password: "",
  congestion_control: "bbr",
  up_mbps: "100",
  down_mbps: "200",
  port_range: "",
}

const XRAY_TRANSPORTS = [
  { value: "tcp", label: "TCP (RAW)" },
  { value: "mkcp", label: "mKCP" },
  { value: "ws", label: "WebSocket" },
  { value: "grpc", label: "gRPC" },
  { value: "httpupgrade", label: "HTTPUpgrade" },
  { value: "xhttp", label: "XHTTP" },
]

const REALITY_TRANSPORTS = [
  { value: "tcp", label: "TCP (RAW)" },
  { value: "xhttp", label: "XHTTP" },
  { value: "grpc", label: "gRPC" },
]

const SING_BOX_TRANSPORTS = [
  { value: "tcp", label: "TCP (RAW)" },
  { value: "ws", label: "WebSocket" },
  { value: "grpc", label: "gRPC" },
]

function isQUICProtocol(protocol: string) {
  return protocol === "hysteria2" || protocol === "tuic"
}

function isShadowsocksProtocol(protocol: string) {
  return protocol === "ss" || protocol === "shadowsocks"
}

function transportOptions(protocol: string, security: string, runtimeType: string) {
  if (isQUICProtocol(protocol) || isShadowsocksProtocol(protocol)) return []
  if (runtimeType === "sing-box" || runtimeType === "singbox") return SING_BOX_TRANSPORTS
  if (protocol === "vless" && security === "reality") return REALITY_TRANSPORTS
  return XRAY_TRANSPORTS
}

function normalizeTransport(protocol: string, security: string, runtimeType: string, transport: string) {
  if (isQUICProtocol(protocol)) return "udp"
  if (isShadowsocksProtocol(protocol)) return "tcp"
  const allowed = transportOptions(protocol, security, runtimeType).map((it) => it.value)
  return allowed.includes(transport) ? transport : "tcp"
}

function isPathHostTransport(transport: string) {
  return transport === "ws" || transport === "httpupgrade" || transport === "xhttp"
}

function transportSectionTitle(transport: string) {
  if (transport === "httpupgrade") return "HTTPUpgrade 参数"
  if (transport === "xhttp") return "XHTTP 参数"
  return "WebSocket 参数"
}

function transportPathLabel(transport: string) {
  if (transport === "httpupgrade") return "HTTPUpgrade Path"
  if (transport === "xhttp") return "XHTTP Path"
  return "WS Path"
}

function transportHostLabel(transport: string) {
  if (transport === "httpupgrade") return "HTTPUpgrade Host"
  if (transport === "xhttp") return "XHTTP Host"
  return "WS Host"
}

function formatTrafficBytes(value: unknown) {
  const bytes = Number(value || 0)
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB", "PB"]
  let n = bytes
  let i = 0
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i += 1
  }
  const fixed = i === 0 ? 0 : n >= 100 ? 0 : n >= 10 ? 1 : 2
  return `${n.toFixed(fixed)} ${units[i]}`
}

// Protocol-specific UI capabilities
function caps(protocol: string, transport: string, security: string) {
  const isQUIC = isQUICProtocol(protocol)
  const isSS = isShadowsocksProtocol(protocol)
  const isVless = protocol === "vless"
  return {
    isQUIC,
    isSS,
    isVless,
    showTransport: !isQUIC && !isSS,
    showSecurity: !isQUIC,
    showPathHost: isPathHostTransport(transport) && !isQUIC,
    showGRPC: transport === "grpc" && !isQUIC,
    showTLS: (security === "tls" || security === "reality" || isQUIC) && !isSS,
    showReality: security === "reality" && isVless,
    showFlow: isVless,
    showALPN: !isSS,
    showMux: !isQUIC && !isSS,
    showSSMethod: isSS,
    showHysteria: protocol === "hysteria2",
    showTUIC: protocol === "tuic",
    showObfs: isQUIC,
    showCongestion: isQUIC,
    showPortRange: protocol === "hysteria2",
  }
}

export default function AdminNodes() {
  const [nodes, setNodes] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [generatingReality, setGeneratingReality] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<any | null>(null)
  const [lastSecret, setLastSecret] = useState("")
  const [secretRevealed, setSecretRevealed] = useState(false)
  const [form, setForm] = useState<NodeForm>(emptyForm)

  const cap = useMemo(() => caps(form.protocol, form.transport, form.security), [form.protocol, form.transport, form.security])
  const currentTransportOptions = useMemo(
    () => transportOptions(form.protocol, form.security, form.runtime_type),
    [form.protocol, form.security, form.runtime_type],
  )

  const load = () => {
    setLoading(true)
    adminGetNodes({ page, pageSize })
      .then((res) => {
        setNodes(res.items || [])
        setTotal(res.total ?? (res.items || []).length)
      })
      .catch((err) => alert(err.message || "加载节点失败"))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [page, pageSize])

  const openCreate = () => {
    setEditing(null)
    setLastSecret("")
    setSecretRevealed(false)
    setForm(emptyForm)
    setDialogOpen(true)
  }

  const openEdit = (n: any) => {
    setEditing(n)
    setLastSecret("")
    setSecretRevealed(false)
    setForm({
      name: n.name || "",
      region: n.region || "",
      host: n.host || "",
      port: String(n.port || 443),
      protocol: n.protocol || "vless",
      transport: n.transport || "tcp",
      security: n.security || "tls",
      runtime_type: n.runtime_type || "xray",
      status: n.status === "inactive" ? "inactive" : "active",
      ws_path: n.ws_path || "/",
      ws_host: n.ws_host || "",
      grpc_service_name: n.grpc_service_name || "",
      sni: n.sni || "",
      fingerprint: n.fingerprint || "chrome",
      reality_public_key: n.reality_public_key || "",
      reality_short_id: n.reality_short_id || "",
      reality_server_name: n.reality_server_name || "",
      reality_private_key: n.reality_private_key || "",
      reality_dest: n.reality_dest || "",
      flow: n.flow || "",
      alpn: n.alpn || "",
      mux_enabled: String(n.mux_enabled || 0),
      ss_method: n.ss_method || "",
      obfs_password: n.obfs_password || "",
      congestion_control: n.congestion_control || "bbr",
      up_mbps: String(n.up_mbps || 100),
      down_mbps: String(n.down_mbps || 200),
      port_range: n.port_range || "",
    })
    setDialogOpen(true)
  }

  const closeDialog = () => {
    if (saving) return
    setDialogOpen(false)
    setEditing(null)
    setLastSecret("")
    setSecretRevealed(false)
    setForm(emptyForm)
  }

  // Auto-pick protocol-safe defaults when protocol changes.
  const onProtocolChange = (protocol: string) => {
    const next = { ...form, protocol }
    if (isQUICProtocol(protocol)) {
      next.runtime_type = "sing-box"
      next.transport = "udp"
      next.security = "tls"
      if (!next.congestion_control) next.congestion_control = "bbr"
    } else if (isShadowsocksProtocol(protocol)) {
      next.security = "none"
      next.transport = "tcp"
      if (!next.ss_method) next.ss_method = "2022-blake3-aes-128-gcm"
    } else {
      if (next.transport === "udp") next.transport = "tcp"
      if (next.security === "none") next.security = "tls"
    }
    next.transport = normalizeTransport(next.protocol, next.security, next.runtime_type, next.transport)
    setForm(next)
  }

  const onSecurityChange = (security: string) => {
    const next = { ...form, security }
    if (form.protocol === "vless" && security === "reality") {
      next.runtime_type = "xray"
    }
    next.transport = normalizeTransport(next.protocol, next.security, next.runtime_type, next.transport)
    setForm(next)
  }

  const onRuntimeChange = (runtimeType: string) => {
    setForm((current) => ({
      ...current,
      runtime_type: runtimeType,
      transport: normalizeTransport(current.protocol, current.security, runtimeType, current.transport),
    }))
  }

  const onTransportChange = (transport: string) => {
    setForm((current) => ({
      ...current,
      transport,
      runtime_type: ["mkcp", "httpupgrade", "xhttp"].includes(transport) ? "xray" : current.runtime_type,
      ws_path: isPathHostTransport(transport) ? current.ws_path || "/" : current.ws_path,
    }))
  }

  const generateReality = async () => {
    setGeneratingReality(true)
    try {
      const serverName = form.reality_server_name.trim() || "www.cloudflare.com"
      const res = await adminGenerateRealityConfig(serverName)
      setForm((current) => ({
        ...current,
        reality_server_name: res.reality_server_name,
        reality_dest: res.reality_dest,
        reality_public_key: res.reality_public_key,
        reality_private_key: res.reality_private_key,
        reality_short_id: res.reality_short_id,
        security: "reality",
        protocol: "vless",
        runtime_type: "xray",
        transport: normalizeTransport("vless", "reality", "xray", current.transport),
      }))
    } catch (err: any) {
      alert(err.message || "生成 Reality 配置失败")
    } finally {
      setGeneratingReality(false)
    }
  }

  const payload = () => ({
    ...form,
    port: Number(form.port || 0),
    mux_enabled: Number(form.mux_enabled || 0),
    up_mbps: Number(form.up_mbps || 0),
    down_mbps: Number(form.down_mbps || 0),
  })

  const save = async () => {
    if (!form.name.trim() || !form.host.trim() || Number(form.port) <= 0) {
      alert("请填写节点名称、地址和端口")
      return
    }
    if (form.protocol === "vless" && form.security === "reality") {
      if (!form.reality_server_name.trim() || !form.reality_public_key.trim() || !form.reality_private_key.trim()) {
        alert("Reality 节点必须填写服务器名、Public Key 和 Private Key")
        return
      }
    }
    setSaving(true)
    try {
      if (editing) {
        await adminUpdateNode(editing.id, payload())
        closeDialog()
      } else {
        const res = await adminCreateNode(payload())
        setLastSecret(res.node_secret)
        setPage(1)
      }
      load()
    } catch (err: any) {
      alert(err.message || "保存失败")
    } finally {
      setSaving(false)
    }
  }

  const handleSync = async (nodeId: number) => {
    try {
      const res = await adminSyncNodeConfig(nodeId)
      alert(`配置同步任务已创建: ${res.task_id}`)
    } catch (err: any) {
      alert(err.message || "同步失败")
    }
  }

  const [syncingAll, setSyncingAll] = useState(false)
  const handleSyncAll = async () => {
    if (syncingAll) return
    if (!confirm("将为所有启用的节点重新下发配置，确认继续？")) return
    setSyncingAll(true)
    try {
      const res = await adminSyncAllNodeConfigs()
      const failed = res.results.filter((r) => r.error)
      let msg = `已下发 ${res.ok}/${res.total} 个节点的同步任务`
      if (failed.length > 0) {
        const detail = failed.slice(0, 5).map((r) => `${r.name}: ${r.error}`).join("\n")
        msg += `\n\n失败 ${failed.length} 个：\n${detail}`
        if (failed.length > 5) msg += `\n…(其余 ${failed.length - 5} 个省略)`
      }
      alert(msg)
      load()
    } catch (err: any) {
      alert(err.message || "批量同步失败")
    } finally {
      setSyncingAll(false)
    }
  }

  const copySecret = async () => {
    if (!lastSecret) return
    await navigator.clipboard.writeText(lastSecret).catch(() => {})
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">节点管理</h1>
          <p className="text-sm text-muted-foreground mt-1">共 {total} 个节点</p>
        </div>
        <div className="flex gap-2">
          <Button size="sm" onClick={load} variant="outline">
            <RefreshCw className="w-4 h-4 mr-1" /> 刷新
          </Button>
          <Button size="sm" onClick={handleSyncAll} variant="outline" disabled={syncingAll || nodes.length === 0}>
            <RefreshCw className={`w-4 h-4 mr-1 ${syncingAll ? "animate-spin" : ""}`} />
            {syncingAll ? "同步中..." : "全部同步"}
          </Button>
          <Button size="sm" onClick={openCreate}>
            <Plus className="w-4 h-4 mr-1" /> 新建节点
          </Button>
        </div>
      </div>

      {nodes.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
          <p className="font-medium">暂无节点</p>
          <p className="text-sm text-muted-foreground mt-1">创建节点后，Agent 可使用节点密钥注册并拉取运行时配置。</p>
          <Button className="mt-4" size="sm" onClick={openCreate}>
            <Plus className="w-4 h-4 mr-1" /> 新建节点
          </Button>
        </div>
      ) : (
        <div className="rounded-lg border bg-card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted/50 border-b">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ID</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">名称</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">健康</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden md:table-cell">地区</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">协议</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">地址</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">在线/流量</th>
                  <th className="text-right px-4 py-3 font-medium text-muted-foreground">操作</th>
                </tr>
              </thead>
              <tbody>
                {nodes.map((n: any) => (
                  <tr key={n.id} className="border-b hover:bg-accent/50">
                    <td className="px-4 py-3">{n.id}</td>
                    <td className="px-4 py-3 font-medium">{n.name}</td>
                    <td className="px-4 py-3">
                      <NodeHealth node={n} />
                    </td>
                    <td className="px-4 py-3 hidden md:table-cell">{n.region || "-"}</td>
                    <td className="px-4 py-3 hidden lg:table-cell">
                      <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded">{n.protocol}</span>
                    </td>
                    <td className="px-4 py-3 hidden lg:table-cell text-muted-foreground">{n.host}:{n.port}</td>
                    <td className="px-4 py-3 hidden lg:table-cell">
                      <div className="leading-tight">
                        <div className="font-medium">{n.active_user_count || 0} 人使用中</div>
                        <div className="mt-1 text-xs text-muted-foreground">
                          {formatTrafficBytes(n.traffic_total)} 总流量
                        </div>
                        <div className="mt-0.5 text-[11px] text-muted-foreground">
                          上 {formatTrafficBytes(n.upload_total)} / 下 {formatTrafficBytes(n.download_total)}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-1">
                        <Button size="icon" variant="ghost" title="编辑" onClick={() => openEdit(n)}>
                          <Edit3 className="w-4 h-4" />
                        </Button>
                        <Button size="sm" variant="ghost" onClick={() => handleSync(n.id)}>
                          <RefreshCw className="w-3 h-3 mr-1" /> 同步
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
      <AdminPager
        page={page}
        pageSize={pageSize}
        total={total}
        onPageChange={setPage}
        onPageSizeChange={(size) => {
          setPageSize(size)
          setPage(1)
        }}
      />

      <Dialog open={dialogOpen} onOpenChange={(open) => { if (!open) closeDialog(); else setDialogOpen(true) }}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto p-0 gap-0">
          <DialogHeader className="px-6 pt-6 pb-4 border-b">
            <DialogTitle className="text-lg">{editing ? "编辑节点" : "新建节点"}</DialogTitle>
            <p className="text-xs text-muted-foreground mt-1">
              {editing
                ? "修改后立即生效，可在节点列表中触发同步使 Agent 拉取最新配置。"
                : "创建后将返回一次性节点密钥，请立即记录并填入 Agent 配置。"}
            </p>
          </DialogHeader>

          {lastSecret ? (
            <div className="px-6 py-6 space-y-4">
              <div className="rounded-xl border-2 border-amber-200 bg-amber-50 p-4">
                <p className="font-semibold text-amber-900">⚠ 节点已创建，请立即保存密钥</p>
                <p className="text-sm text-amber-800 mt-1">该密钥只返回一次，关闭对话框后无法再次查看。</p>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">节点密钥 (node_secret)</Label>
                <div className="mt-1.5 flex items-center gap-2">
                  <Input
                    readOnly
                    value={secretRevealed ? lastSecret : "•".repeat(Math.min(lastSecret.length, 48))}
                    className="font-mono text-xs h-10"
                  />
                  <Button
                    size="icon"
                    variant="outline"
                    onClick={() => setSecretRevealed(!secretRevealed)}
                    title={secretRevealed ? "隐藏" : "显示"}
                  >
                    {secretRevealed ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </Button>
                  <Button size="icon" variant="outline" onClick={copySecret} title="复制密钥">
                    <Copy className="w-4 h-4" />
                  </Button>
                </div>
              </div>
              <div className="flex justify-end pt-2">
                <Button onClick={closeDialog}>完成</Button>
              </div>
            </div>
          ) : (
            <>
              <div className="px-6 py-5 space-y-6">
                {/* Section: 基础信息 */}
                <Section title="基础信息" icon={<Server className="w-4 h-4" />}>
                  <Row cols={2}>
                    <Field label="节点名称" required hint="显示在订阅与管理页">
                      <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="HK-01" className="h-10" />
                    </Field>
                    <Field label="地区" hint="可选 · 显示用">
                      <Input value={form.region} onChange={(e) => setForm({ ...form, region: e.target.value })} placeholder="HK / JP / US" className="h-10" />
                    </Field>
                  </Row>
                  <Row cols={3} template="1fr 110px 130px">
                    <Field label="服务器地址" required>
                      <Input value={form.host} onChange={(e) => setForm({ ...form, host: e.target.value })} placeholder="example.com 或 IP" className="h-10" />
                    </Field>
                    <Field label="端口" required>
                      <Input type="number" min="1" max="65535" value={form.port} onChange={(e) => setForm({ ...form, port: e.target.value })} className="h-10" />
                    </Field>
                    <Field label="状态">
                      <Select value={form.status} onValueChange={(status: "active" | "inactive") => setForm({ ...form, status })}>
                        <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="active">启用</SelectItem>
                          <SelectItem value="inactive">停用</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                  </Row>
                </Section>

                {/* Section: 协议配置 */}
                <Section title="协议配置" icon={<Network className="w-4 h-4" />}>
                  <Row cols={2}>
                    <Field label="协议" required>
                      <Select value={form.protocol} onValueChange={onProtocolChange}>
                        <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="vless">VLESS</SelectItem>
                          <SelectItem value="vmess">VMess</SelectItem>
                          <SelectItem value="trojan">Trojan</SelectItem>
                          <SelectItem value="ss">Shadowsocks</SelectItem>
                          <SelectItem value="hysteria2">Hysteria2</SelectItem>
                          <SelectItem value="tuic">TUIC</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                    <Field label="运行时" hint={cap.isQUIC ? "QUIC 协议仅 sing-box 支持" : ""}>
                      <Select value={form.runtime_type} onValueChange={onRuntimeChange} disabled={cap.isQUIC}>
                        <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="xray">Xray</SelectItem>
                          <SelectItem value="sing-box">sing-box</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                  </Row>
                  {cap.showTransport && (
                    <Row cols={2}>
                      <Field label="传输方式">
                        <Select value={form.transport} onValueChange={onTransportChange}>
                          <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                          <SelectContent>
                            {currentTransportOptions.map((it) => (
                              <SelectItem key={it.value} value={it.value}>{it.label}</SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </Field>
                      {cap.showSecurity && (
                        <Field label="加密层">
                          <Select value={form.security} onValueChange={onSecurityChange}>
                            <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                            <SelectContent>
                              <SelectItem value="tls">TLS</SelectItem>
                              {cap.isVless && <SelectItem value="reality">Reality</SelectItem>}
                              <SelectItem value="none">None</SelectItem>
                            </SelectContent>
                          </Select>
                        </Field>
                      )}
                    </Row>
                  )}
                </Section>

                {/* Section: TLS / Reality */}
                {cap.showTLS && (
                  <Section title={cap.showReality ? "Reality 设置" : "TLS 设置"} icon={<Lock className="w-4 h-4" />}>
                    {!cap.showReality && (
                      <Row cols={2}>
                        <Field label="SNI" hint="留空则使用服务器地址">
                          <Input value={form.sni} onChange={(e) => setForm({ ...form, sni: e.target.value })} placeholder={form.host || "example.com"} className="h-10" />
                        </Field>
                        <Field label="Fingerprint">
                          <Select value={form.fingerprint || "none"} onValueChange={(v) => setForm({ ...form, fingerprint: v === "none" ? "" : v })}>
                            <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                            <SelectContent>
                              <SelectItem value="none">默认</SelectItem>
                              <SelectItem value="chrome">chrome</SelectItem>
                              <SelectItem value="firefox">firefox</SelectItem>
                              <SelectItem value="safari">safari</SelectItem>
                              <SelectItem value="edge">edge</SelectItem>
                            </SelectContent>
                          </Select>
                        </Field>
                      </Row>
                    )}
                    {cap.showReality && (
                      <>
                        <div className="flex justify-end">
                          <Button type="button" size="sm" variant="outline" onClick={generateReality} disabled={generatingReality}>
                            {generatingReality ? <RefreshCw className="w-4 h-4 mr-1 animate-spin" /> : <Wand2 className="w-4 h-4 mr-1" />}
                            自动生成
                          </Button>
                        </div>
                        <Row cols={2}>
                          <Field label="Reality 服务器名" required hint="客户端连接时使用">
                            <Input value={form.reality_server_name} onChange={(e) => setForm({ ...form, reality_server_name: e.target.value })} placeholder="www.cloudflare.com" className="h-10" />
                          </Field>
                          <Field label="Reality 目标" hint="服务端伪装回源地址">
                            <Input value={form.reality_dest} onChange={(e) => setForm({ ...form, reality_dest: e.target.value })} placeholder="www.cloudflare.com:443" className="h-10" />
                          </Field>
                        </Row>
                        <Row cols={2}>
                          <Field label="Reality Public Key" required>
                            <Input value={form.reality_public_key} onChange={(e) => setForm({ ...form, reality_public_key: e.target.value })} className="h-10 font-mono text-xs" placeholder="客户端使用" />
                          </Field>
                          <Field label="Reality Private Key" required hint="服务端使用，不会下发到订阅">
                            <Input type="password" value={form.reality_private_key} onChange={(e) => setForm({ ...form, reality_private_key: e.target.value })} className="h-10 font-mono text-xs" />
                          </Field>
                        </Row>
                        <Row cols={2}>
                          <Field label="Short ID">
                            <Input value={form.reality_short_id} onChange={(e) => setForm({ ...form, reality_short_id: e.target.value })} className="h-10 font-mono text-xs" placeholder="可选 · 短标识" />
                          </Field>
                          <Field label="Fingerprint">
                            <Select value={form.fingerprint || "none"} onValueChange={(v) => setForm({ ...form, fingerprint: v === "none" ? "" : v })}>
                              <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                              <SelectContent>
                                <SelectItem value="none">默认</SelectItem>
                                <SelectItem value="chrome">chrome</SelectItem>
                                <SelectItem value="firefox">firefox</SelectItem>
                                <SelectItem value="safari">safari</SelectItem>
                                <SelectItem value="edge">edge</SelectItem>
                              </SelectContent>
                            </Select>
                          </Field>
                        </Row>
                      </>
                    )}
                  </Section>
                )}

                {/* Section: path/host transports */}
                {cap.showPathHost && (
                  <Section title={transportSectionTitle(form.transport)} icon={<Network className="w-4 h-4" />}>
                    <Row cols={2}>
                      <Field label={transportPathLabel(form.transport)}>
                        <Input value={form.ws_path} onChange={(e) => setForm({ ...form, ws_path: e.target.value })} placeholder="/" className="h-10" />
                      </Field>
                      <Field label={transportHostLabel(form.transport)} hint="留空则使用 SNI">
                        <Input value={form.ws_host} onChange={(e) => setForm({ ...form, ws_host: e.target.value })} placeholder="cdn.example.com" className="h-10" />
                      </Field>
                    </Row>
                  </Section>
                )}

                {/* Section: gRPC */}
                {cap.showGRPC && (
                  <Section title="gRPC 参数" icon={<Network className="w-4 h-4" />}>
                    <Field label="Service Name" required>
                      <Input value={form.grpc_service_name} onChange={(e) => setForm({ ...form, grpc_service_name: e.target.value })} placeholder="grpc-service" className="h-10" />
                    </Field>
                  </Section>
                )}

                {/* Section: Hysteria2 / TUIC */}
                {cap.isQUIC && (
                  <Section title={cap.showHysteria ? "Hysteria2 参数" : "TUIC 参数"} icon={<Sliders className="w-4 h-4" />}>
                    <Row cols={2}>
                      <Field label="SNI" hint="留空则使用服务器地址">
                        <Input value={form.sni} onChange={(e) => setForm({ ...form, sni: e.target.value })} placeholder={form.host || "example.com"} className="h-10" />
                      </Field>
                      <Field label="拥塞控制">
                        <Select value={form.congestion_control} onValueChange={(v) => setForm({ ...form, congestion_control: v })}>
                          <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                          <SelectContent>
                            <SelectItem value="bbr">BBR</SelectItem>
                            <SelectItem value="cubic">CUBIC</SelectItem>
                            <SelectItem value="new_reno">New Reno</SelectItem>
                          </SelectContent>
                        </Select>
                      </Field>
                    </Row>
                    {cap.showHysteria && (
                      <>
                        <Row cols={2}>
                          <Field label="上行 Mbps">
                            <Input type="number" min="0" value={form.up_mbps} onChange={(e) => setForm({ ...form, up_mbps: e.target.value })} className="h-10" />
                          </Field>
                          <Field label="下行 Mbps">
                            <Input type="number" min="0" value={form.down_mbps} onChange={(e) => setForm({ ...form, down_mbps: e.target.value })} className="h-10" />
                          </Field>
                        </Row>
                        <Row cols={2}>
                          <Field label="Salamander 混淆密码" hint="留空则不启用混淆">
                            <Input value={form.obfs_password} onChange={(e) => setForm({ ...form, obfs_password: e.target.value })} className="h-10" />
                          </Field>
                          <Field label="端口跳跃" hint="格式 20000-40000，需 root 权限">
                            <Input value={form.port_range} onChange={(e) => setForm({ ...form, port_range: e.target.value })} placeholder="20000-40000" className="h-10" />
                          </Field>
                        </Row>
                      </>
                    )}
                    {cap.showTUIC && (
                      <Field label="共享密码" hint="留空则使用每用户 client_id">
                        <Input value={form.obfs_password} onChange={(e) => setForm({ ...form, obfs_password: e.target.value })} className="h-10" />
                      </Field>
                    )}
                  </Section>
                )}

                {/* Section: Shadowsocks */}
                {cap.isSS && (
                  <Section title="Shadowsocks 参数" icon={<Lock className="w-4 h-4" />}>
                    <Field label="加密方式">
                      <Select value={form.ss_method || "2022-blake3-aes-128-gcm"} onValueChange={(v) => setForm({ ...form, ss_method: v })}>
                        <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="2022-blake3-aes-128-gcm">2022-blake3-aes-128-gcm</SelectItem>
                          <SelectItem value="2022-blake3-aes-256-gcm">2022-blake3-aes-256-gcm</SelectItem>
                          <SelectItem value="2022-blake3-chacha20-poly1305">2022-blake3-chacha20-poly1305</SelectItem>
                          <SelectItem value="aes-128-gcm">aes-128-gcm</SelectItem>
                          <SelectItem value="aes-256-gcm">aes-256-gcm</SelectItem>
                          <SelectItem value="chacha20-ietf-poly1305">chacha20-ietf-poly1305</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                  </Section>
                )}

                {/* Section: 高级 */}
                {(cap.showFlow || cap.showALPN || cap.showMux) && (
                  <Section title="高级选项" icon={<Sliders className="w-4 h-4" />} collapsible>
                    {cap.showFlow && cap.isVless && (
                      <Field label="VLESS Flow" hint="未选择则不下发 flow">
                        <Select value={form.flow || "none"} onValueChange={(v) => setForm({ ...form, flow: v === "none" ? "" : v })}>
                          <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                          <SelectContent>
                            <SelectItem value="none">无</SelectItem>
                            <SelectItem value="xtls-rprx-vision">xtls-rprx-vision</SelectItem>
                          </SelectContent>
                        </Select>
                      </Field>
                    )}
                    <Row cols={2}>
                      {cap.showALPN && (
                        <Field label="ALPN" hint="逗号分隔，如 h2,http/1.1">
                          <Input value={form.alpn} onChange={(e) => setForm({ ...form, alpn: e.target.value })} placeholder="h2,http/1.1" className="h-10" />
                        </Field>
                      )}
                      {cap.showMux && (
                        <Field label="多路复用 (Mux)">
                          <Select value={form.mux_enabled} onValueChange={(v) => setForm({ ...form, mux_enabled: v })}>
                            <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                            <SelectContent>
                              <SelectItem value="0">关闭</SelectItem>
                              <SelectItem value="1">开启</SelectItem>
                            </SelectContent>
                          </Select>
                        </Field>
                      )}
                    </Row>
                  </Section>
                )}
              </div>
              <div className="px-6 py-4 border-t bg-muted/30 flex justify-end gap-2">
                <Button variant="outline" onClick={closeDialog} disabled={saving}>取消</Button>
                <Button onClick={save} disabled={saving} className="min-w-24">
                  {saving ? "保存中..." : editing ? "保存修改" : "创建节点"}
                </Button>
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}

function Section({ title, icon, children, collapsible = false }: { title: string; icon?: ReactNode; children: ReactNode; collapsible?: boolean }) {
  const [open, setOpen] = useState(!collapsible)
  return (
    <section className="rounded-lg border bg-card/50">
      <button
        type="button"
        onClick={() => collapsible && setOpen(!open)}
        className={`w-full flex items-center gap-2 px-4 py-2.5 text-left ${collapsible ? "cursor-pointer hover:bg-accent/50" : "cursor-default"}`}
        disabled={!collapsible}
      >
        {icon && <span className="text-primary">{icon}</span>}
        <h3 className="text-sm font-semibold flex-1">{title}</h3>
        {collapsible && (
          <span className="text-xs text-muted-foreground">{open ? "收起" : "展开"}</span>
        )}
      </button>
      {open && <div className="px-4 pb-4 space-y-3 border-t pt-3">{children}</div>}
    </section>
  )
}

function NodeHealth({ node }: { node: any }) {
  const status = node.health_status || (node.status === "active" ? "yellow" : "red")
  const palette: Record<string, string> = {
    green: "bg-green-500 text-green-700",
    yellow: "bg-yellow-400 text-yellow-700",
    red: "bg-red-500 text-red-700",
  }
  const color = palette[status] || palette.red
  const lastSeen = node.last_heartbeat_at ? new Date(node.last_heartbeat_at).toLocaleString("zh-CN") : "无心跳"

  return (
    <div className="flex items-center gap-2">
      <span className={`h-2.5 w-2.5 rounded-full ${color.split(" ")[0]}`} aria-hidden="true" />
      <div className="leading-tight">
        <div className={`text-xs font-medium ${color.split(" ")[1]}`}>
          {node.health_label || "异常"}
        </div>
        <div className="text-[11px] text-muted-foreground">
          {node.runtime_status || "-"} · {lastSeen}
        </div>
      </div>
    </div>
  )
}

function Row({ cols, template, children }: { cols?: number; template?: string; children: ReactNode }) {
  const style = template ? { gridTemplateColumns: template } : undefined
  const className = template ? "grid gap-3" : `grid gap-3 grid-cols-1 sm:grid-cols-${cols || 2}`
  return <div className={className} style={style}>{children}</div>
}

function Field({ label, required, hint, children }: { label: string; required?: boolean; hint?: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium">
        {label}
        {required && <span className="text-red-500 ml-0.5">*</span>}
      </Label>
      {children}
      {hint && <p className="text-[11px] text-muted-foreground">{hint}</p>}
    </div>
  )
}
