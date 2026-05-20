"use client"

import { useEffect, useState } from "react"
import type { ReactNode } from "react"
import { Copy, Edit3, Plus, RefreshCw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { adminCreateNode, adminGetNodes, adminSyncNodeConfig, adminUpdateNode } from "@/lib/api"

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

export default function AdminNodes() {
  const [nodes, setNodes] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<any | null>(null)
  const [lastSecret, setLastSecret] = useState("")
  const [form, setForm] = useState<NodeForm>(emptyForm)

  const load = () => {
    setLoading(true)
    adminGetNodes()
      .then((res) => setNodes(res.items || []))
      .catch((err) => alert(err.message || "加载节点失败"))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [])

  const openCreate = () => {
    setEditing(null)
    setLastSecret("")
    setForm(emptyForm)
    setDialogOpen(true)
  }

  const openEdit = (n: any) => {
    setEditing(n)
    setLastSecret("")
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
    setForm(emptyForm)
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
    setSaving(true)
    try {
      if (editing) {
        await adminUpdateNode(editing.id, payload())
        closeDialog()
      } else {
        const res = await adminCreateNode(payload())
        setLastSecret(res.node_secret)
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
          <p className="text-sm text-muted-foreground mt-1">共 {nodes.length} 个节点</p>
        </div>
        <div className="flex gap-2">
          <Button size="sm" onClick={load} variant="outline">
            <RefreshCw className="w-4 h-4 mr-1" /> 刷新
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
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden md:table-cell">地区</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">协议</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">地址</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">状态</th>
                  <th className="text-right px-4 py-3 font-medium text-muted-foreground">操作</th>
                </tr>
              </thead>
              <tbody>
                {nodes.map((n: any) => (
                  <tr key={n.id} className="border-b hover:bg-accent/50">
                    <td className="px-4 py-3">{n.id}</td>
                    <td className="px-4 py-3 font-medium">{n.name}</td>
                    <td className="px-4 py-3 hidden md:table-cell">{n.region || "-"}</td>
                    <td className="px-4 py-3 hidden lg:table-cell">
                      <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded">{n.protocol}</span>
                    </td>
                    <td className="px-4 py-3 hidden lg:table-cell text-muted-foreground">{n.host}:{n.port}</td>
                    <td className="px-4 py-3">
                      <span className={`text-xs rounded-full px-2 py-1 ${
                        n.status === "active" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-600"
                      }`}>
                        {n.status === "active" ? "启用" : "停用"}
                      </span>
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

      <Dialog open={dialogOpen} onOpenChange={(open) => { if (!open) closeDialog(); else setDialogOpen(true) }}>
        <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editing ? "编辑节点" : "新建节点"}</DialogTitle>
          </DialogHeader>

          {lastSecret ? (
            <div className="rounded-lg border bg-muted/40 p-4 space-y-3">
              <div>
                <p className="font-medium">节点已创建，请立即保存密钥</p>
                <p className="text-sm text-muted-foreground mt-1">该密钥只返回一次，Agent 注册时需要使用。</p>
              </div>
              <div className="flex items-center gap-2">
                <Input readOnly value={lastSecret} className="font-mono text-xs" />
                <Button size="icon" variant="outline" onClick={copySecret} title="复制密钥">
                  <Copy className="w-4 h-4" />
                </Button>
              </div>
              <div className="flex justify-end">
                <Button onClick={closeDialog}>完成</Button>
              </div>
            </div>
          ) : (
            <>
              <div className="grid gap-5 py-2">
                <section className="grid gap-3">
                  <h3 className="text-sm font-semibold">基础信息</h3>
                  <div className="grid grid-cols-2 gap-3">
                    <Field label="节点名称">
                      <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
                    </Field>
                    <Field label="地区">
                      <Input value={form.region} onChange={(e) => setForm({ ...form, region: e.target.value })} placeholder="HK / JP / US" />
                    </Field>
                  </div>
                  <div className="grid grid-cols-[1fr_120px_140px] gap-3">
                    <Field label="地址">
                      <Input value={form.host} onChange={(e) => setForm({ ...form, host: e.target.value })} />
                    </Field>
                    <Field label="端口">
                      <Input type="number" min="1" max="65535" value={form.port} onChange={(e) => setForm({ ...form, port: e.target.value })} />
                    </Field>
                    <Field label="状态">
                      <Select value={form.status} onValueChange={(status: "active" | "inactive") => setForm({ ...form, status })}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="active">启用</SelectItem>
                          <SelectItem value="inactive">停用</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                  </div>
                </section>

                <section className="grid gap-3">
                  <h3 className="text-sm font-semibold">协议与运行时</h3>
                  <div className="grid grid-cols-4 gap-3">
                    <Field label="协议">
                      <Select value={form.protocol} onValueChange={(protocol) => setForm({ ...form, protocol })}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="vless">VLESS</SelectItem>
                          <SelectItem value="vmess">VMess</SelectItem>
                          <SelectItem value="ss">Shadowsocks</SelectItem>
                          <SelectItem value="trojan">Trojan</SelectItem>
                          <SelectItem value="hysteria2">Hysteria2</SelectItem>
                          <SelectItem value="tuic">TUIC</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                    <Field label="传输">
                      <Input value={form.transport} onChange={(e) => setForm({ ...form, transport: e.target.value })} />
                    </Field>
                    <Field label="安全">
                      <Input value={form.security} onChange={(e) => setForm({ ...form, security: e.target.value })} />
                    </Field>
                    <Field label="Runtime">
                      <Select value={form.runtime_type} onValueChange={(runtime_type) => setForm({ ...form, runtime_type })}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="xray">Xray</SelectItem>
                          <SelectItem value="sing-box">sing-box</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                  </div>
                </section>

                <section className="grid gap-3">
                  <h3 className="text-sm font-semibold">TLS / Reality / 传输参数</h3>
                  <div className="grid grid-cols-3 gap-3">
                    <Field label="SNI">
                      <Input value={form.sni} onChange={(e) => setForm({ ...form, sni: e.target.value })} />
                    </Field>
                    <Field label="Fingerprint">
                      <Input value={form.fingerprint} onChange={(e) => setForm({ ...form, fingerprint: e.target.value })} />
                    </Field>
                    <Field label="Flow">
                      <Input value={form.flow} onChange={(e) => setForm({ ...form, flow: e.target.value })} />
                    </Field>
                  </div>
                  <div className="grid grid-cols-3 gap-3">
                    <Field label="WS Path">
                      <Input value={form.ws_path} onChange={(e) => setForm({ ...form, ws_path: e.target.value })} />
                    </Field>
                    <Field label="WS Host">
                      <Input value={form.ws_host} onChange={(e) => setForm({ ...form, ws_host: e.target.value })} />
                    </Field>
                    <Field label="gRPC Service">
                      <Input value={form.grpc_service_name} onChange={(e) => setForm({ ...form, grpc_service_name: e.target.value })} />
                    </Field>
                  </div>
                  <div className="grid grid-cols-3 gap-3">
                    <Field label="Reality Public Key">
                      <Input value={form.reality_public_key} onChange={(e) => setForm({ ...form, reality_public_key: e.target.value })} />
                    </Field>
                    <Field label="Reality Short ID">
                      <Input value={form.reality_short_id} onChange={(e) => setForm({ ...form, reality_short_id: e.target.value })} />
                    </Field>
                    <Field label="Reality Server Name">
                      <Input value={form.reality_server_name} onChange={(e) => setForm({ ...form, reality_server_name: e.target.value })} />
                    </Field>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <Field label="Reality Private Key">
                      <Input value={form.reality_private_key} onChange={(e) => setForm({ ...form, reality_private_key: e.target.value })} />
                    </Field>
                    <Field label="Reality Dest">
                      <Input value={form.reality_dest} onChange={(e) => setForm({ ...form, reality_dest: e.target.value })} placeholder="www.cloudflare.com:443" />
                    </Field>
                  </div>
                </section>

                <section className="grid gap-3">
                  <h3 className="text-sm font-semibold">高级参数</h3>
                  <div className="grid grid-cols-4 gap-3">
                    <Field label="ALPN">
                      <Input value={form.alpn} onChange={(e) => setForm({ ...form, alpn: e.target.value })} placeholder="h2,http/1.1" />
                    </Field>
                    <Field label="Mux">
                      <Select value={form.mux_enabled} onValueChange={(mux_enabled) => setForm({ ...form, mux_enabled })}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="0">关闭</SelectItem>
                          <SelectItem value="1">开启</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                    <Field label="SS Method">
                      <Input value={form.ss_method} onChange={(e) => setForm({ ...form, ss_method: e.target.value })} />
                    </Field>
                    <Field label="端口跳跃">
                      <Input value={form.port_range} onChange={(e) => setForm({ ...form, port_range: e.target.value })} placeholder="20000-40000" />
                    </Field>
                  </div>
                  <div className="grid grid-cols-4 gap-3">
                    <Field label="Obfs Password">
                      <Input value={form.obfs_password} onChange={(e) => setForm({ ...form, obfs_password: e.target.value })} />
                    </Field>
                    <Field label="拥塞控制">
                      <Input value={form.congestion_control} onChange={(e) => setForm({ ...form, congestion_control: e.target.value })} />
                    </Field>
                    <Field label="上行 Mbps">
                      <Input type="number" min="0" value={form.up_mbps} onChange={(e) => setForm({ ...form, up_mbps: e.target.value })} />
                    </Field>
                    <Field label="下行 Mbps">
                      <Input type="number" min="0" value={form.down_mbps} onChange={(e) => setForm({ ...form, down_mbps: e.target.value })} />
                    </Field>
                  </div>
                </section>
              </div>
              <div className="flex justify-end gap-2">
                <Button variant="outline" onClick={closeDialog} disabled={saving}>取消</Button>
                <Button onClick={save} disabled={saving}>{saving ? "保存中..." : "保存"}</Button>
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}
