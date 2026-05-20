"use client"

import { useEffect, useMemo, useState } from "react"
import type { ReactNode } from "react"
import { Edit3, Plus, Power, PowerOff } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { adminCreateUser, adminGetUsers, adminUpdateUser } from "@/lib/api"

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

export default function AdminUsers() {
  const [users, setUsers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<any | null>(null)
  const [form, setForm] = useState<UserForm>(emptyForm)

  const isEditing = useMemo(() => Boolean(editing), [editing])

  const load = () => {
    setLoading(true)
    adminGetUsers()
      .then((res) => setUsers(res.items || []))
      .catch((err) => alert(err.message || "加载用户失败"))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [])

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

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">用户管理</h1>
          <p className="text-sm text-muted-foreground mt-1">共 {users.length} 个用户</p>
        </div>
        <Button size="sm" onClick={openCreate}>
          <Plus className="w-4 h-4 mr-1" /> 新建用户
        </Button>
      </div>

      {users.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
          <p className="font-medium">暂无用户</p>
          <p className="text-sm text-muted-foreground mt-1">可以直接创建测试账号或为客户开通账号。</p>
          <Button className="mt-4" size="sm" onClick={openCreate}>
            <Plus className="w-4 h-4 mr-1" /> 新建用户
          </Button>
        </div>
      ) : (
        <div className="rounded-lg border bg-card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted/50 border-b">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ID</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">邮箱</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden md:table-cell">流量</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">到期时间</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">状态</th>
                  <th className="text-right px-4 py-3 font-medium text-muted-foreground">操作</th>
                </tr>
              </thead>
              <tbody>
                {users.map((u: any) => (
                  <tr key={u.id} className="border-b hover:bg-accent/50">
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
                        <Button size="icon" variant="ghost" title="编辑" onClick={() => openEdit(u)}>
                          <Edit3 className="w-4 h-4" />
                        </Button>
                        <Button size="icon" variant="ghost" title={u.status === "active" ? "禁用" : "启用"} onClick={() => toggleStatus(u)}>
                          {u.status === "active" ? <PowerOff className="w-4 h-4" /> : <Power className="w-4 h-4" />}
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
              <Field label="套餐 ID">
                <Input value={form.plan_id} onChange={(e) => setForm({ ...form, plan_id: e.target.value })} placeholder="可留空" />
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
