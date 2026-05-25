"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import type { ReactNode } from "react"
import {
  Ban,
  Copy,
  Edit3,
  FileText,
  Filter,
  History,
  KeyRound,
  Mail,
  MoreHorizontal,
  PackagePlus,
  Plus,
  Power,
  PowerOff,
  RotateCcw,
  Search,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Textarea } from "@/components/ui/textarea"
import { AdminPager } from "@/components/admin/AdminPager"
import {
  adminBatchUsers,
  adminCreateUser,
  adminGetPlans,
  adminGetUserSubscription,
  adminGetUserOrders,
  adminGetUsers,
  adminGetUserTrafficLogs,
  adminResetUserIdentity,
  adminUpdateUser,
  buildSubscriptionUrl,
  getPublicSettings,
} from "@/lib/api"

type UserForm = {
  email: string
  password: string
  balance: string
  plan_id: string
  expired_at: string
  traffic_limit_gb: string
  traffic_used_gb: string
  status: "active" | "disabled"
}

type UserFilter = {
  email: string
  status: string
  plan_id: string
  expires: string
  traffic_min_gb: string
  traffic_max_gb: string
}

const emptyForm: UserForm = {
  email: "",
  password: "",
  balance: "0.00",
  plan_id: "",
  expired_at: "",
  traffic_limit_gb: "0",
  traffic_used_gb: "0",
  status: "active",
}

const emptyFilter: UserFilter = {
  email: "",
  status: "all",
  plan_id: "all",
  expires: "all",
  traffic_min_gb: "",
  traffic_max_gb: "",
}

export default function AdminUsers() {
  const [users, setUsers] = useState<any[]>([])
  const [plans, setPlans] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [mailOpen, setMailOpen] = useState(false)
  const [filterOpen, setFilterOpen] = useState(false)
  const [detailOpen, setDetailOpen] = useState(false)
  const [editing, setEditing] = useState<any | null>(null)
  const [detailTitle, setDetailTitle] = useState("")
  const [detailItems, setDetailItems] = useState<any[]>([])
  const [form, setForm] = useState<UserForm>(emptyForm)
  const [filters, setFilters] = useState<UserFilter>(emptyFilter)
  const [settings, setSettings] = useState<Record<string, string>>({})
  const [mailSubject, setMailSubject] = useState("")
  const [mailContent, setMailContent] = useState("")
  const [selected, setSelected] = useState<number[]>([])

  const isEditing = useMemo(() => Boolean(editing), [editing])
  const planMap = useMemo(() => {
    const m = new Map<number, any>()
    for (const p of plans) m.set(Number(p.id), p)
    return m
  }, [plans])

  const load = useCallback(() => {
    setLoading(true)
    Promise.all([
      adminGetUsers({
        page,
        pageSize,
        email: filters.email.trim(),
        status: filters.status,
        planId: filters.plan_id,
        expires: filters.expires,
        trafficMin: filters.traffic_min_gb ? gbToBytes(filters.traffic_min_gb) : null,
        trafficMax: filters.traffic_max_gb ? gbToBytes(filters.traffic_max_gb) : null,
      }),
      adminGetPlans({ page: 1, pageSize: 100 }),
      getPublicSettings().catch(() => ({ settings: {} })),
    ])
      .then(([u, p, publicSettings]) => {
        setUsers(u.items || [])
        setTotal(u.total ?? (u.items || []).length)
        setPlans(p.items || [])
        setSettings(publicSettings.settings || {})
        setSelected([])
      })
      .catch((err) => alert(err.message || "加载失败"))
      .finally(() => setLoading(false))
  }, [filters, page, pageSize])

  useEffect(() => { load() }, [load])

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setDialogOpen(true)
  }

  const openEdit = (u: any) => {
    setEditing(u)
    setForm({
      email: u.email || "",
      password: "",
      balance: u.balance || "0.00",
      plan_id: u.plan_id ? String(u.plan_id) : "",
      expired_at: u.expired_at ? String(u.expired_at).slice(0, 10) : "",
      traffic_limit_gb: bytesToGB(u.traffic_limit),
      traffic_used_gb: bytesToGB(u.traffic_used),
      status: u.status === "disabled" ? "disabled" : "active",
    })
    setDialogOpen(true)
  }

  const closeDialog = () => {
    if (saving) return
    setDialogOpen(false)
    setEditing(null)
    setForm(emptyForm)
  }

  const payload = () => ({
    email: form.email.trim(),
    ...(isEditing ? {} : { password: form.password }),
    balance: form.balance.trim() || "0.00",
    plan_id: form.plan_id ? Number(form.plan_id) : null,
    expired_at: form.expired_at || null,
    traffic_limit: gbToBytes(form.traffic_limit_gb),
    traffic_used: gbToBytes(form.traffic_used_gb),
    status: form.status,
  })

  const save = async () => {
    if (!form.email.trim()) {
      alert("请输入邮箱")
      return
    }
    if (!isEditing && form.password.length < 6) {
      alert("初始密码至少 6 位")
      return
    }
    setSaving(true)
    try {
      if (isEditing) {
        await adminUpdateUser(editing.id, payload())
      } else {
        await adminCreateUser(payload())
        setPage(1)
      }
      closeDialog()
      load()
    } catch (err: any) {
      alert(err.message || "保存失败")
    } finally {
      setSaving(false)
    }
  }

  const toggleStatus = async (u: any) => {
    const next = u.status === "active" ? "disabled" : "active"
    try {
      await adminUpdateUser(u.id, {
        email: u.email,
        balance: u.balance || "0.00",
        plan_id: u.plan_id,
        expired_at: u.expired_at ? String(u.expired_at).slice(0, 10) : null,
        traffic_limit: u.traffic_limit || 0,
        traffic_used: u.traffic_used || 0,
        status: next,
      })
      load()
    } catch (err: any) {
      alert(err.message || "状态更新失败")
    }
  }

  const selectedOnPage = users.length > 0 && users.every((u) => selected.includes(u.id))
  const toggleSelectAll = () => {
    if (selectedOnPage) {
      setSelected((prev) => prev.filter((id) => !users.some((u) => u.id === id)))
      return
    }
    setSelected((prev) => Array.from(new Set([...prev, ...users.map((u) => u.id)])))
  }

  const toggleSelect = (id: number) => {
    setSelected((prev) => prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id])
  }

  const runBatch = async (action: "enable" | "disable" | "reset_subscription") => {
    if (selected.length === 0) {
      alert("请先选择用户")
      return
    }
    const label = action === "enable" ? "启用" : action === "disable" ? "禁用" : "重置订阅 URL"
    if (!confirm(`确认对 ${selected.length} 个用户执行「${label}」？`)) return
    try {
      await adminBatchUsers(action, selected)
      load()
    } catch (err: any) {
      alert(err.message || "批量操作失败")
    }
  }

  const sendBatchEmail = async () => {
    if (selected.length === 0) {
      alert("请先选择用户")
      return
    }
    if (!mailSubject.trim() || !mailContent.trim()) {
      alert("请填写邮件标题和内容")
      return
    }
    try {
      await adminBatchUsers("send_email", selected, { subject: mailSubject.trim(), content: mailContent.trim() })
      setMailOpen(false)
      setMailSubject("")
      setMailContent("")
      alert("邮件发送任务已完成")
    } catch (err: any) {
      alert(err.message || "发送邮件失败")
    }
  }

  const exportCSV = () => {
    const rows = selected.length > 0 ? users.filter((u) => selected.includes(u.id)) : users
    const csv = [
      ["id", "email", "status", "plan_id", "expired_at", "traffic_used", "traffic_limit"],
      ...rows.map((u) => [
        u.id,
        u.email,
        u.status,
        u.plan_id || "",
        u.expired_at || "",
        u.traffic_used || 0,
        u.traffic_limit || 0,
      ]),
    ].map((row) => row.map(csvCell).join(",")).join("\n")
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = `zboard-users-${new Date().toISOString().slice(0, 10)}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const copySubscription = async (u: any) => {
    try {
      const res = await adminGetUserSubscription(u.id)
      const url = buildSubscriptionUrl(res.token, undefined, settings)
      await navigator.clipboard.writeText(url)
      alert("已复制订阅 URL")
    } catch (err: any) {
      alert(err.message || "复制订阅 URL 失败")
    }
  }

  const resetUserIdentity = async (u: any) => {
    if (!confirm("重置 UUID 及订阅 URL 会让旧客户端和旧订阅链接失效，确认继续？")) return
    try {
      const res = await adminResetUserIdentity(u.id)
      const url = buildSubscriptionUrl(res.token, undefined, settings)
      await navigator.clipboard.writeText(url)
      alert("UUID 和订阅 URL 已重置，新的订阅 URL 已复制")
      load()
    } catch (err: any) {
      alert(err.message || "重置身份失败")
    }
  }

  const showOrders = async (u: any) => {
    try {
      const res = await adminGetUserOrders(u.id)
      setDetailTitle(`${u.email} 的订单`)
      setDetailItems(res.items || [])
      setDetailOpen(true)
    } catch (err: any) {
      alert(err.message || "加载订单失败")
    }
  }

  const showTrafficLogs = async (u: any) => {
    try {
      const res = await adminGetUserTrafficLogs(u.id)
      setDetailTitle(`${u.email} 的流量记录`)
      setDetailItems(res.items || [])
      setDetailOpen(true)
    } catch (err: any) {
      alert(err.message || "加载流量记录失败")
    }
  }

  const applyFilters = () => {
    setPage(1)
    setFilterOpen(false)
    load()
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-2xl font-bold">用户管理</h1>
          <p className="text-sm text-muted-foreground mt-1">共 {total} 个用户，已选择 {selected.length} 个</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="outline" onClick={() => setFilterOpen(true)}>
            <Filter className="w-4 h-4 mr-1" /> 过滤器
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="outline">
                <MoreHorizontal className="w-4 h-4 mr-1" /> 批量操作
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
              <DropdownMenuItem onClick={exportCSV}>
                <FileText className="w-4 h-4" /> 导出 CSV
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setMailOpen(true)} disabled={selected.length === 0}>
                <Mail className="w-4 h-4" /> 发送邮件
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => runBatch("enable")}>
                <Power className="w-4 h-4" /> 批量启用
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => runBatch("disable")}>
                <Ban className="w-4 h-4" /> 批量禁用
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => runBatch("reset_subscription")}>
                <RotateCcw className="w-4 h-4" /> 重置订阅 URL
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button size="sm" onClick={openCreate}>
            <Plus className="w-4 h-4 mr-1" /> 新建用户
          </Button>
        </div>
      </div>

      <div className="rounded-lg border bg-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-muted/50 border-b">
                <th className="w-10 px-4 py-3">
                  <Checkbox checked={selectedOnPage} onCheckedChange={toggleSelectAll} aria-label="选择当前页" />
                </th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">ID</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">邮箱</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden md:table-cell">流量</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">到期时间</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">状态</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">操作</th>
              </tr>
            </thead>
            <tbody>
              {users.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center text-muted-foreground">没有匹配的用户</td>
                </tr>
              ) : users.map((u: any) => (
                <tr key={u.id} className="border-b hover:bg-accent/50">
                  <td className="px-4 py-3">
                    <Checkbox checked={selected.includes(u.id)} onCheckedChange={() => toggleSelect(u.id)} aria-label={`选择 ${u.email}`} />
                  </td>
                  <td className="px-4 py-3">{u.id}</td>
                  <td className="px-4 py-3 font-medium">{u.email}</td>
                  <td className="px-4 py-3 hidden md:table-cell text-muted-foreground">
                    {bytesToGB(u.traffic_used)} / {bytesToGB(u.traffic_limit)} GB
                  </td>
                  <td className="px-4 py-3 hidden lg:table-cell text-muted-foreground">
                    {u.expired_at ? new Date(u.expired_at).toLocaleDateString("zh-CN") : "-"}
                  </td>
                  <td className="px-4 py-3">
                    <span className={`text-xs rounded-full px-2 py-1 ${
                      u.status === "active" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"
                    }`}>
                      {u.status === "active" ? "正常" : "已禁用"}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex justify-end gap-1">
                      <Button size="icon" variant="ghost" title={u.status === "active" ? "禁用" : "启用"} onClick={() => toggleStatus(u)}>
                        {u.status === "active" ? <PowerOff className="w-4 h-4" /> : <Power className="w-4 h-4" />}
                      </Button>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button size="icon" variant="ghost" title="更多操作">
                            <MoreHorizontal className="w-4 h-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-56">
                          <DropdownMenuItem onClick={() => openEdit(u)}><Edit3 className="w-4 h-4" /> 编辑</DropdownMenuItem>
                          <DropdownMenuItem onClick={() => openEdit(u)}><PackagePlus className="w-4 h-4" /> 分配订单/套餐</DropdownMenuItem>
                          <DropdownMenuItem onClick={() => copySubscription(u)}><Copy className="w-4 h-4" /> 复制订阅 URL</DropdownMenuItem>
                          <DropdownMenuItem onClick={() => resetUserIdentity(u)}><KeyRound className="w-4 h-4" /> 重置 UUID 及订阅 URL</DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem onClick={() => showOrders(u)}><FileText className="w-4 h-4" /> TA 的订单</DropdownMenuItem>
                          <DropdownMenuItem onClick={() => showTrafficLogs(u)}><History className="w-4 h-4" /> TA 的流量记录</DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

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

      <Sheet open={filterOpen} onOpenChange={setFilterOpen}>
        <SheetContent className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>过滤器</SheetTitle>
          </SheetHeader>
          <div className="space-y-4 px-4">
            <Field label="邮箱">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input className="pl-9" value={filters.email} onChange={(e) => setFilters({ ...filters, email: e.target.value })} placeholder="输入邮箱关键字" />
              </div>
            </Field>
            <Field label="状态">
              <Select value={filters.status} onValueChange={(status) => setFilters({ ...filters, status })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部</SelectItem>
                  <SelectItem value="active">正常</SelectItem>
                  <SelectItem value="disabled">已禁用</SelectItem>
                </SelectContent>
              </Select>
            </Field>
            <Field label="套餐">
              <Select value={filters.plan_id} onValueChange={(plan_id) => setFilters({ ...filters, plan_id })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部套餐</SelectItem>
                  {plans.map((p: any) => <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>)}
                </SelectContent>
              </Select>
            </Field>
            <Field label="到期状态">
              <Select value={filters.expires} onValueChange={(expires) => setFilters({ ...filters, expires })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部</SelectItem>
                  <SelectItem value="valid">未到期</SelectItem>
                  <SelectItem value="expired">已到期</SelectItem>
                </SelectContent>
              </Select>
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="最小已用 GB">
                <Input type="number" min="0" value={filters.traffic_min_gb} onChange={(e) => setFilters({ ...filters, traffic_min_gb: e.target.value })} />
              </Field>
              <Field label="最大已用 GB">
                <Input type="number" min="0" value={filters.traffic_max_gb} onChange={(e) => setFilters({ ...filters, traffic_max_gb: e.target.value })} />
              </Field>
            </div>
          </div>
          <SheetFooter>
            <div className="flex gap-2">
              <Button variant="outline" className="flex-1" onClick={() => { setFilters(emptyFilter); setPage(1) }}>重置</Button>
              <Button className="flex-1" onClick={applyFilters}>检索</Button>
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <Dialog open={dialogOpen} onOpenChange={(open) => { if (!open) closeDialog(); else setDialogOpen(true) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{isEditing ? "编辑用户" : "新建用户"}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <Field label="邮箱">
              <Input value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
            </Field>
            {!isEditing && (
              <Field label="初始密码">
                <Input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
              </Field>
            )}
            <div className="grid grid-cols-2 gap-3">
              <Field label="状态">
                <Select value={form.status} onValueChange={(status: "active" | "disabled") => setForm({ ...form, status })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">正常</SelectItem>
                    <SelectItem value="disabled">禁用</SelectItem>
                  </SelectContent>
                </Select>
              </Field>
              <Field label="余额">
                <Input value={form.balance} onChange={(e) => setForm({ ...form, balance: e.target.value })} />
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <Field label="订阅套餐">
                <Select
                  value={form.plan_id || "none"}
                  onValueChange={(v) => {
                    if (v === "none") {
                      setForm({ ...form, plan_id: "" })
                      return
                    }
                    const plan = planMap.get(Number(v))
                    if (!plan) {
                      setForm({ ...form, plan_id: v })
                      return
                    }
                    const days = Number(plan.duration_days || 0)
                    const expire = days > 0
                      ? new Date(Date.now() + days * 86400_000).toISOString().slice(0, 10)
                      : form.expired_at
                    setForm({ ...form, plan_id: v, expired_at: expire, traffic_limit_gb: bytesToGB(plan.traffic_limit) })
                  }}
                >
                  <SelectTrigger><SelectValue placeholder="未分配" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">未分配</SelectItem>
                    {plans.map((p: any) => (
                      <SelectItem key={p.id} value={String(p.id)}>
                        {p.name} · {bytesToGB(p.traffic_limit)}GB · {p.duration_days}天
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
              <Field label="到期日期">
                <Input type="date" value={form.expired_at} onChange={(e) => setForm({ ...form, expired_at: e.target.value })} />
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <Field label="流量上限 GB">
                <Input type="number" min="0" value={form.traffic_limit_gb} onChange={(e) => setForm({ ...form, traffic_limit_gb: e.target.value })} />
              </Field>
              <Field label="已用流量 GB">
                <Input type="number" min="0" value={form.traffic_used_gb} onChange={(e) => setForm({ ...form, traffic_used_gb: e.target.value })} />
              </Field>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={closeDialog} disabled={saving}>取消</Button>
            <Button onClick={save} disabled={saving}>{saving ? "保存中..." : "保存"}</Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={mailOpen} onOpenChange={setMailOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>发送邮件</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">将发送给已选择的 {selected.length} 个用户。</p>
            <Field label="邮件标题">
              <Input value={mailSubject} onChange={(e) => setMailSubject(e.target.value)} />
            </Field>
            <Field label="邮件内容">
              <Textarea rows={6} value={mailContent} onChange={(e) => setMailContent(e.target.value)} />
            </Field>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setMailOpen(false)}>取消</Button>
            <Button onClick={sendBatchEmail}>发送</Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={detailOpen} onOpenChange={setDetailOpen}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader><DialogTitle>{detailTitle}</DialogTitle></DialogHeader>
          <Textarea readOnly rows={14} value={JSON.stringify(detailItems, null, 2)} className="font-mono text-xs" />
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

function bytesToGB(value: number | null | undefined) {
  return String(Number(((value || 0) / 1073741824).toFixed(1)).toString())
}

function gbToBytes(value: string) {
  return Math.max(0, Math.round(Number(value || 0) * 1073741824))
}

function csvCell(value: any) {
  const raw = String(value ?? "")
  return `"${raw.replace(/"/g, '""')}"`
}
