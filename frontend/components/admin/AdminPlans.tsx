"use client"

import { useEffect, useState } from "react"
import type { ReactNode } from "react"
import { Edit3, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { adminCreatePlan, adminGetPlans, adminUpdatePlan } from "@/lib/api"

type PlanForm = {
  name: string
  price: string
  duration_days: string
  traffic_limit_gb: string
  device_limit: string
  speed_limit: string
  status: "active" | "inactive"
  sort: string
}

const emptyForm: PlanForm = {
  name: "",
  price: "9.90",
  duration_days: "30",
  traffic_limit_gb: "100",
  device_limit: "3",
  speed_limit: "0",
  status: "active",
  sort: "0",
}

export default function AdminPlans() {
  const [plans, setPlans] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<any | null>(null)
  const [form, setForm] = useState<PlanForm>(emptyForm)

  const load = () => {
    setLoading(true)
    adminGetPlans()
      .then((res) => setPlans(res.items || []))
      .catch((err) => alert(err.message || "加载套餐失败"))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [])

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setDialogOpen(true)
  }

  const openEdit = (p: any) => {
    setEditing(p)
    setForm({
      name: p.name || "",
      price: p.price || "0.00",
      duration_days: String(p.duration_days || 30),
      traffic_limit_gb: bytesToGB(p.traffic_limit),
      device_limit: String(p.device_limit || 3),
      speed_limit: String(p.speed_limit || 0),
      status: p.status === "inactive" ? "inactive" : "active",
      sort: String(p.sort || 0),
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
    name: form.name.trim(),
    price: form.price.trim(),
    duration_days: Number(form.duration_days || 0),
    traffic_limit: gbToBytes(form.traffic_limit_gb),
    device_limit: Number(form.device_limit || 0),
    speed_limit: Number(form.speed_limit || 0),
    status: form.status,
    sort: Number(form.sort || 0),
  })

  const save = async () => {
    if (!form.name.trim() || !form.price.trim() || Number(form.duration_days) <= 0) {
      alert("请填写套餐名称、价格和有效天数")
      return
    }
    setSaving(true)
    try {
      if (editing) {
        await adminUpdatePlan(editing.id, payload())
      } else {
        await adminCreatePlan(payload())
      }
      closeDialog()
      load()
    } catch (err: any) {
      alert(err.message || "保存失败")
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">套餐管理</h1>
          <p className="text-sm text-muted-foreground mt-1">共 {plans.length} 个套餐</p>
        </div>
        <Button size="sm" onClick={openCreate}>
          <Plus className="w-4 h-4 mr-1" /> 新建套餐
        </Button>
      </div>

      {plans.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
          <p className="font-medium">暂无套餐</p>
          <p className="text-sm text-muted-foreground mt-1">先创建一个套餐，用户端才能下单购买。</p>
          <Button className="mt-4" size="sm" onClick={openCreate}>
            <Plus className="w-4 h-4 mr-1" /> 新建套餐
          </Button>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {plans.map((p: any) => (
            <div key={p.id} className="rounded-lg border bg-card p-5">
              <div className="flex items-center justify-between gap-3 mb-2">
                <h3 className="font-semibold truncate">{p.name}</h3>
                <span className={`text-xs rounded-full px-2 py-0.5 ${
                  p.status === "active" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-600"
                }`}>
                  {p.status === "active" ? "上架" : "下架"}
                </span>
              </div>
              <div className="text-2xl font-bold">¥{p.price}</div>
              <div className="text-sm text-muted-foreground mt-1">{p.duration_days} 天</div>
              <div className="mt-3 text-xs text-muted-foreground space-y-1">
                <p>流量: {bytesToGB(p.traffic_limit)} GB</p>
                <p>设备: {p.device_limit} 台</p>
                <p>排序: {p.sort}</p>
              </div>
              <div className="mt-4 flex justify-end">
                <Button size="sm" variant="outline" onClick={() => openEdit(p)}>
                  <Edit3 className="w-4 h-4 mr-1" /> 编辑
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Dialog open={dialogOpen} onOpenChange={(open) => { if (!open) closeDialog(); else setDialogOpen(true) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? "编辑套餐" : "新建套餐"}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <Field label="套餐名称">
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="价格">
                <Input value={form.price} onChange={(e) => setForm({ ...form, price: e.target.value })} />
              </Field>
              <Field label="状态">
                <Select value={form.status} onValueChange={(status: "active" | "inactive") => setForm({ ...form, status })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">上架</SelectItem>
                    <SelectItem value="inactive">下架</SelectItem>
                  </SelectContent>
                </Select>
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <Field label="有效天数">
                <Input type="number" min="1" value={form.duration_days} onChange={(e) => setForm({ ...form, duration_days: e.target.value })} />
              </Field>
              <Field label="流量 GB">
                <Input type="number" min="0" value={form.traffic_limit_gb} onChange={(e) => setForm({ ...form, traffic_limit_gb: e.target.value })} />
              </Field>
            </div>
            <div className="grid grid-cols-3 gap-3">
              <Field label="设备数">
                <Input type="number" min="1" value={form.device_limit} onChange={(e) => setForm({ ...form, device_limit: e.target.value })} />
              </Field>
              <Field label="限速 Mbps">
                <Input type="number" min="0" value={form.speed_limit} onChange={(e) => setForm({ ...form, speed_limit: e.target.value })} />
              </Field>
              <Field label="排序">
                <Input type="number" value={form.sort} onChange={(e) => setForm({ ...form, sort: e.target.value })} />
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
