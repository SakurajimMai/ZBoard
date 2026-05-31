"use client"

import { useCallback, useEffect, useState } from "react"
import type { ReactNode } from "react"
import { Bell, Edit3, Megaphone, Plus, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { AdminPager } from "@/components/admin/AdminPager"
import { toast } from "sonner"
import { useConfirm } from "@/components/confirm-dialog"
import {
  adminCreateAnnouncement,
  adminDeleteAnnouncement,
  adminGetAnnouncements,
  adminUpdateAnnouncement,
} from "@/lib/api"

type FormState = {
  title: string
  content: string
  popup: boolean
  priority: string
  status: "active" | "inactive"
  starts_at: string
  ends_at: string
}

const emptyForm: FormState = {
  title: "",
  content: "",
  popup: true,
  priority: "0",
  status: "active",
  starts_at: "",
  ends_at: "",
}

function toDateTimeLocal(value?: string | null): string {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ""
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function fromDateTimeLocal(value: string): string | null {
  if (!value) return null
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null
  return date.toISOString()
}

export default function AdminAnnouncements() {
  const confirm = useConfirm()
  const [items, setItems] = useState<any[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<any | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm)

  const load = useCallback(() => {
    setLoading(true)
    adminGetAnnouncements({ page, pageSize })
      .then((res) => {
        setItems(res.items || [])
        setTotal(res.total ?? (res.items || []).length)
      })
      .catch((err) => toast.error(err.message || "加载公告失败"))
      .finally(() => setLoading(false))
  }, [page, pageSize])

  useEffect(() => { load() }, [load])

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setDialogOpen(true)
  }

  const openEdit = (item: any) => {
    setEditing(item)
    setForm({
      title: item.title || "",
      content: item.content || "",
      popup: Boolean(item.popup),
      priority: String(item.priority ?? 0),
      status: item.status === "inactive" ? "inactive" : "active",
      starts_at: toDateTimeLocal(item.starts_at),
      ends_at: toDateTimeLocal(item.ends_at),
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
    title: form.title.trim(),
    content: form.content.trim(),
    popup: form.popup,
    priority: Number(form.priority || 0),
    status: form.status,
    starts_at: fromDateTimeLocal(form.starts_at),
    ends_at: fromDateTimeLocal(form.ends_at),
  })

  const save = async () => {
    if (!form.title.trim() || !form.content.trim()) {
      toast.error("请填写标题和内容")
      return
    }
    setSaving(true)
    try {
      if (editing) {
        await adminUpdateAnnouncement(editing.id, payload())
        toast.success("公告已更新")
      } else {
        await adminCreateAnnouncement(payload())
        toast.success("公告已创建")
        setPage(1)
      }
      closeDialog()
      load()
    } catch (err: any) {
      toast.error(err.message || "保存公告失败")
    } finally {
      setSaving(false)
    }
  }

  const remove = async (item: any) => {
    if (!(await confirm({ title: "删除公告", description: `确认删除公告「${item.title}」？`, destructive: true, confirmText: "删除" }))) return
    try {
      await adminDeleteAnnouncement(item.id)
      toast.success("公告已删除")
      load()
    } catch (err: any) {
      toast.error(err.message || "删除公告失败")
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">公告管理</h1>
          <p className="text-sm text-muted-foreground mt-1">管理用户端公告和弹窗通知</p>
        </div>
        <Button size="sm" onClick={openCreate}>
          <Plus className="w-4 h-4 mr-1" /> 添加公告
        </Button>
      </div>

      <div className="rounded-lg border bg-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-muted/50 border-b">
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">标题</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">展示</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">弹窗</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">优先级</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden md:table-cell">创建时间</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">操作</th>
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-16 text-center text-muted-foreground">暂无公告</td>
                </tr>
              ) : items.map((item) => (
                <tr key={item.id} className="border-b hover:bg-accent/50">
                  <td className="px-4 py-3">
                    <div className="font-medium">{item.title}</div>
                    <div className="text-xs text-muted-foreground line-clamp-1">{item.content}</div>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`text-xs rounded-full px-2 py-1 ${item.status === "active" ? "bg-green-100 text-green-700" : "bg-zinc-100 text-zinc-600"}`}>
                      {item.status === "active" ? "显示" : "隐藏"}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    {item.popup ? <Bell className="w-4 h-4 text-primary" /> : <Megaphone className="w-4 h-4 text-muted-foreground" />}
                  </td>
                  <td className="px-4 py-3">{item.priority}</td>
                  <td className="px-4 py-3 hidden md:table-cell text-muted-foreground">
                    {item.created_at ? new Date(item.created_at).toLocaleString("zh-CN") : "-"}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex justify-end gap-1">
                      <Button size="icon" variant="ghost" onClick={() => openEdit(item)} title="编辑">
                        <Edit3 className="w-4 h-4" />
                      </Button>
                      <Button size="icon" variant="ghost" onClick={() => remove(item)} title="删除">
                        <Trash2 className="w-4 h-4" />
                      </Button>
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

      <Dialog open={dialogOpen} onOpenChange={(open) => { if (!open) closeDialog(); else setDialogOpen(true) }}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{editing ? "编辑公告" : "添加公告"}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <Field label="标题">
              <Input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
            </Field>
            <Field label="内容">
              <Textarea rows={5} value={form.content} onChange={(e) => setForm({ ...form, content: e.target.value })} />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="状态">
                <Select value={form.status} onValueChange={(status: "active" | "inactive") => setForm({ ...form, status })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">显示</SelectItem>
                    <SelectItem value="inactive">隐藏</SelectItem>
                  </SelectContent>
                </Select>
              </Field>
              <Field label="优先级">
                <Input type="number" value={form.priority} onChange={(e) => setForm({ ...form, priority: e.target.value })} />
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <Field label="开始时间">
                <Input type="datetime-local" value={form.starts_at} onChange={(e) => setForm({ ...form, starts_at: e.target.value })} />
              </Field>
              <Field label="结束时间">
                <Input type="datetime-local" value={form.ends_at} onChange={(e) => setForm({ ...form, ends_at: e.target.value })} />
              </Field>
            </div>
            <div className="flex items-center justify-between rounded-lg border p-3">
              <div>
                <p className="text-sm font-medium">弹窗通知</p>
                <p className="text-xs text-muted-foreground">用户进入控制台后弹窗展示一次</p>
              </div>
              <Switch checked={form.popup} onCheckedChange={(popup) => setForm({ ...form, popup })} />
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
