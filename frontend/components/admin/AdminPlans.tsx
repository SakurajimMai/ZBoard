"use client"

import { useEffect, useState } from "react"
import type { ReactNode } from "react"
import { Check, Edit3, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { AdminPager } from "@/components/admin/AdminPager"
import { adminCreatePlan, adminGetPlans, adminUpdatePlan } from "@/lib/api"

type PlanForm = {
  name: string
  price: string
  quarterly_price: string
  yearly_price: string
  reset_traffic_price: string
  duration_days: string
  traffic_limit_gb: string
  device_limit: string
  features_text: string
  status: "active" | "inactive"
  sort: string
}

const emptyForm: PlanForm = {
  name: "",
  price: "9.90",
  quarterly_price: "26.90",
  yearly_price: "99.00",
  reset_traffic_price: "0.00",
  duration_days: "30",
  traffic_limit_gb: "100",
  device_limit: "3",
  features_text: "",
  status: "active",
  sort: "0",
}

export default function AdminPlans() {
  const [plans, setPlans] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(12)
  const [total, setTotal] = useState(0)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<any | null>(null)
  const [form, setForm] = useState<PlanForm>(emptyForm)

  const load = () => {
    setLoading(true)
    adminGetPlans({ page, pageSize })
      .then((res) => {
        setPlans(res.items || [])
        setTotal(res.total ?? (res.items || []).length)
      })
      .catch((err) => alert(err.message || "加载套餐失败"))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [page, pageSize])

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
      quarterly_price: p.quarterly_price || "0.00",
      yearly_price: p.yearly_price || "0.00",
      reset_traffic_price: p.reset_traffic_price || "0.00",
      duration_days: String(p.duration_days || 30),
      traffic_limit_gb: bytesToGB(p.traffic_limit),
      device_limit: String(p.device_limit || 3),
      features_text: featureList(p).join("\n"),
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
    quarterly_price: form.quarterly_price.trim() || "0.00",
    yearly_price: form.yearly_price.trim() || "0.00",
    reset_traffic_price: form.reset_traffic_price.trim() || "0.00",
    duration_days: Number(form.duration_days || 0),
    traffic_limit: gbToBytes(form.traffic_limit_gb),
    device_limit: Number(form.device_limit || 0),
    features: form.features_text.split("\n").map((it) => it.trim()).filter(Boolean),
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

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">套餐管理</h1>
          <p className="text-sm text-muted-foreground mt-1">共 {total} 个套餐</p>
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
              <div className="grid grid-cols-3 gap-2 mt-3">
                <PricePill label="月付" price={p.price} />
                <PricePill label="季付" price={periodPrice(p, "quarterly")} />
                <PricePill label="年付" price={periodPrice(p, "yearly")} />
              </div>
              <div className="text-sm text-muted-foreground mt-3">月周期 {p.duration_days} 天，季付/年付自动折算周期与流量</div>
              <div className="mt-3 text-xs text-muted-foreground space-y-1">
                <p>流量: {bytesToGB(p.traffic_limit)} GB</p>
                <p>设备: {p.device_limit} 台</p>
                <p>排序: {p.sort}</p>
              </div>
              <ul className="mt-4 space-y-2 text-sm">
                {featureList(p).map((feature) => (
                  <li key={feature} className="flex items-start gap-2 text-muted-foreground">
                    <Check className="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
                    <span>{feature}</span>
                  </li>
                ))}
              </ul>
              <div className="mt-4 flex justify-end">
                <Button size="sm" variant="outline" onClick={() => openEdit(p)}>
                  <Edit3 className="w-4 h-4 mr-1" /> 编辑
                </Button>
              </div>
            </div>
          ))}
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
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? "编辑套餐" : "新建套餐"}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <Field label="套餐名称">
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="月付价格">
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
              <Field label="季付价格">
                <Input type="number" min="0" step="0.01" value={form.quarterly_price} onChange={(e) => setForm({ ...form, quarterly_price: e.target.value })} />
              </Field>
              <Field label="年付价格">
                <Input type="number" min="0" step="0.01" value={form.yearly_price} onChange={(e) => setForm({ ...form, yearly_price: e.target.value })} />
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
            <div className="grid grid-cols-2 gap-3">
              <Field label="设备数">
                <Input type="number" min="1" value={form.device_limit} onChange={(e) => setForm({ ...form, device_limit: e.target.value })} />
              </Field>
              <Field label="排序">
                <Input type="number" value={form.sort} onChange={(e) => setForm({ ...form, sort: e.target.value })} />
              </Field>
            </div>
            <Field label="重置流量单价（CNY，0 表示禁用）">
              <Input
                type="number"
                min="0"
                step="0.01"
                value={form.reset_traffic_price}
                onChange={(e) => setForm({ ...form, reset_traffic_price: e.target.value })}
                placeholder="0.00"
              />
            </Field>
            <Field label="自定义卖点">
              <Textarea
                value={form.features_text}
                onChange={(e) => setForm({ ...form, features_text: e.target.value })}
                placeholder="每行一条，例如：&#10;100 GB 流量&#10;3 台设备同时在线&#10;全部节点可用&#10;支持 Clash / sing-box / V2rayN"
                className="min-h-28"
              />
            </Field>
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

function PricePill({ label, price }: { label: string; price: string }) {
  return (
    <div className="rounded-md border bg-secondary/30 px-3 py-2">
      <div className="text-[11px] text-muted-foreground">{label}</div>
      <div className="text-sm font-semibold">¥{price}</div>
    </div>
  )
}

function periodPrice(plan: any, period: "quarterly" | "yearly") {
  const monthly = Number(plan.price || 0)
  const value = period === "quarterly" ? Number(plan.quarterly_price || 0) : Number(plan.yearly_price || 0)
  if (value > 0) return value.toFixed(2)
  return (monthly * (period === "quarterly" ? 3 : 12)).toFixed(2)
}

function bytesToGB(value: number | null | undefined) {
  return String(Number(((value || 0) / 1073741824).toFixed(1)).toString())
}

function gbToBytes(value: string) {
  return Math.max(0, Math.round(Number(value || 0) * 1073741824))
}

function featureList(plan: any): string[] {
  const explicit = Array.isArray(plan.features) ? plan.features.filter(Boolean) : []
  if (explicit.length > 0) return explicit
  const list = [
    `${bytesToGB(plan.traffic_limit)} GB 流量`,
    `${plan.device_limit || 3} 台设备同时在线`,
  ]
  list.push("全部节点可用", "支持 Clash / sing-box / V2rayN")
  return list
}
