"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import type { ReactNode } from "react"
import { BookOpen, Edit3, Eye, GripVertical, Pencil, Plus, Search, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { AdminPager } from "@/components/admin/AdminPager"
import MarkdownRenderer from "@/components/MarkdownRenderer"
import {
  adminCreateKnowledge,
  adminDeleteKnowledge,
  adminGetKnowledge,
  adminUpdateKnowledge,
} from "@/lib/api"

type FormState = {
  title: string
  category: string
  summary: string
  content: string
  sort: string
  status: "active" | "inactive"
}

const emptyForm: FormState = {
  title: "",
  category: "通用教程",
  summary: "",
  content: "",
  sort: "0",
  status: "active",
}

export default function AdminKnowledge() {
  const [items, setItems] = useState<any[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<any | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm)
  const [categoryFilter, setCategoryFilter] = useState("all")
  const [statusFilter, setStatusFilter] = useState("all")
  const [keyword, setKeyword] = useState("")
  const [editorTab, setEditorTab] = useState<"edit" | "preview">("edit")

  const load = useCallback(() => {
    setLoading(true)
    adminGetKnowledge({
      page,
      pageSize,
      category: categoryFilter === "all" ? "" : categoryFilter,
      status: statusFilter === "all" ? "" : statusFilter,
    })
      .then((res) => {
        setItems(res.items || [])
        setTotal(res.total ?? (res.items || []).length)
      })
      .catch((err) => alert(err.message || "加载知识库失败"))
      .finally(() => setLoading(false))
  }, [page, pageSize, categoryFilter, statusFilter])

  useEffect(() => { load() }, [load])

  const categories = useMemo(() => {
    const values = new Set(items.map((item) => item.category).filter(Boolean))
    return Array.from(values)
  }, [items])

  const visibleItems = useMemo(() => {
    const q = keyword.trim().toLowerCase()
    if (!q) return items
    return items.filter((item) => `${item.title} ${item.category} ${item.summary}`.toLowerCase().includes(q))
  }, [items, keyword])

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setEditorTab("edit")
    setDialogOpen(true)
  }

  const openEdit = (item: any) => {
    setEditing(item)
    setForm({
      title: item.title || "",
      category: item.category || "通用教程",
      summary: item.summary || "",
      content: item.content || "",
      sort: String(item.sort ?? 0),
      status: item.status === "inactive" ? "inactive" : "active",
    })
    setEditorTab("edit")
    setDialogOpen(true)
  }

  const closeDialog = () => {
    if (saving) return
    setDialogOpen(false)
    setEditing(null)
    setForm(emptyForm)
  }

  const payload = (source = form) => ({
    title: source.title.trim(),
    category: source.category.trim() || "通用教程",
    summary: source.summary.trim(),
    content: source.content.trim(),
    sort: Number(source.sort || 0),
    status: source.status,
  })

  const save = async () => {
    if (!form.title.trim() || !form.content.trim()) {
      alert("请填写标题和教程内容")
      return
    }
    setSaving(true)
    try {
      if (editing) {
        await adminUpdateKnowledge(editing.id, payload())
      } else {
        await adminCreateKnowledge(payload())
        setPage(1)
      }
      closeDialog()
      load()
    } catch (err: any) {
      alert(err.message || "保存教程失败")
    } finally {
      setSaving(false)
    }
  }

  const toggleVisible = async (item: any, checked: boolean) => {
    try {
      await adminUpdateKnowledge(item.id, {
        title: item.title,
        category: item.category,
        summary: item.summary || "",
        content: item.content,
        sort: item.sort || 0,
        status: checked ? "active" : "inactive",
      })
      load()
    } catch (err: any) {
      alert(err.message || "更新显示状态失败")
    }
  }

  const remove = async (item: any) => {
    if (!confirm(`确认删除教程「${item.title}」？`)) return
    try {
      await adminDeleteKnowledge(item.id)
      load()
    } catch (err: any) {
      alert(err.message || "删除教程失败")
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">知识库管理</h1>
          <p className="text-sm text-muted-foreground mt-1">维护用户端使用教程、客户端导入指南和常见配置说明</p>
        </div>
        <Button size="sm" onClick={openCreate}>
          <Plus className="w-4 h-4 mr-1" /> 新增教程
        </Button>
      </div>

      <div className="flex flex-col gap-3 rounded-lg border bg-card p-4 md:flex-row md:items-center">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input value={keyword} onChange={(e) => setKeyword(e.target.value)} placeholder="搜索标题、分类或摘要" className="pl-9" />
        </div>
        <Select value={categoryFilter} onValueChange={(v) => { setCategoryFilter(v); setPage(1) }}>
          <SelectTrigger className="w-full md:w-44"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部分类</SelectItem>
            {categories.map((category) => <SelectItem key={category} value={category}>{category}</SelectItem>)}
          </SelectContent>
        </Select>
        <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(1) }}>
          <SelectTrigger className="w-full md:w-36"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="active">显示</SelectItem>
            <SelectItem value="inactive">隐藏</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-lg border bg-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-muted/50 border-b">
                <th className="w-12 px-4 py-3 text-left font-medium text-muted-foreground">排序</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">文章ID</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">显示</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">标题</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">分类</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground hidden md:table-cell">更新时间</th>
                <th className="px-4 py-3 text-right font-medium text-muted-foreground">操作</th>
              </tr>
            </thead>
            <tbody>
              {visibleItems.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center text-muted-foreground">暂无教程</td>
                </tr>
              ) : visibleItems.map((item) => (
                <tr key={item.id} className="border-b hover:bg-accent/50">
                  <td className="px-4 py-3 text-muted-foreground">
                    <div className="flex items-center gap-2">
                      <GripVertical className="h-4 w-4" />
                      <span>{item.sort}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{item.id}</td>
                  <td className="px-4 py-3">
                    <Switch checked={item.status === "active"} onCheckedChange={(checked) => toggleVisible(item, checked)} />
                  </td>
                  <td className="px-4 py-3">
                    <div className="font-medium">{item.title}</div>
                    <div className="max-w-md truncate text-xs text-muted-foreground">{item.summary || item.slug}</div>
                  </td>
                  <td className="px-4 py-3">{item.category || "通用教程"}</td>
                  <td className="px-4 py-3 hidden md:table-cell text-muted-foreground">
                    {item.updated_at ? new Date(item.updated_at).toLocaleString("zh-CN") : "-"}
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
        <DialogContent className="sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>{editing ? "编辑教程" : "新增教程"}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <Field label="标题">
              <Input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
            </Field>
            <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
              <Field label="分类">
                <Input value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} placeholder="Windows / Android / iOS / Mac" />
              </Field>
              <Field label="排序">
                <Input type="number" value={form.sort} onChange={(e) => setForm({ ...form, sort: e.target.value })} />
              </Field>
              <Field label="状态">
                <Select value={form.status} onValueChange={(status: "active" | "inactive") => setForm({ ...form, status })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">显示</SelectItem>
                    <SelectItem value="inactive">隐藏</SelectItem>
                  </SelectContent>
                </Select>
              </Field>
            </div>
            <Field label="摘要">
              <Input value={form.summary} onChange={(e) => setForm({ ...form, summary: e.target.value })} placeholder="用于用户端列表展示" />
            </Field>

            {/* 编辑 / 预览切换 */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>教程内容</Label>
                <div className="flex rounded-lg border bg-muted p-0.5">
                  <button
                    type="button"
                    onClick={() => setEditorTab("edit")}
                    className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${editorTab === "edit" ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"}`}
                  >
                    <Pencil className="h-3.5 w-3.5" /> 编辑
                  </button>
                  <button
                    type="button"
                    onClick={() => setEditorTab("preview")}
                    className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${editorTab === "preview" ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"}`}
                  >
                    <Eye className="h-3.5 w-3.5" /> 预览
                  </button>
                </div>
              </div>

              {editorTab === "edit" ? (
                <Textarea
                  className="min-h-80 font-mono text-sm"
                  value={form.content}
                  onChange={(e) => setForm({ ...form, content: e.target.value })}
                  placeholder={"支持 Markdown 语法，例如：\n\n## 标题\n\n**粗体** 和 *斜体*\n\n- 列表项一\n- 列表项二\n\n```\n代码块\n```"}
                />
              ) : (
                <div className="min-h-80 rounded-md border bg-card p-5 overflow-y-auto">
                  {form.content.trim() ? (
                    <MarkdownRenderer content={form.content} />
                  ) : (
                    <p className="text-sm text-muted-foreground">暂无内容可以预览，请切换到编辑模式输入 Markdown 内容</p>
                  )}
                </div>
              )}
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
