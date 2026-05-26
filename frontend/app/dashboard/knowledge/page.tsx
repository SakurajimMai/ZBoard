"use client"

import { useEffect, useMemo, useState } from "react"
import { BookOpen, ChevronRight, Search } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import MarkdownRenderer from "@/components/MarkdownRenderer"
import { getKnowledge, getKnowledgeArticle } from "@/lib/api"

export default function KnowledgePage() {
  const [items, setItems] = useState<any[]>([])
  const [selected, setSelected] = useState<any | null>(null)
  const [category, setCategory] = useState("all")
  const [keyword, setKeyword] = useState("")
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    getKnowledge(category === "all" ? undefined : category)
      .then((res) => {
        const list = res.items || []
        setItems(list)
        if (list.length > 0) {
          return getKnowledgeArticle(list[0].slug).then((detail) => setSelected(detail.article))
        }
        setSelected(null)
      })
      .catch(() => {
        setItems([])
        setSelected(null)
      })
      .finally(() => setLoading(false))
  }, [category])

  const categories = useMemo(() => {
    const values = new Set(items.map((item) => item.category).filter(Boolean))
    return Array.from(values)
  }, [items])

  const filtered = useMemo(() => {
    const q = keyword.trim().toLowerCase()
    if (!q) return items
    return items.filter((item) => `${item.title} ${item.category} ${item.summary}`.toLowerCase().includes(q))
  }, [items, keyword])

  const openArticle = async (item: any) => {
    setSelected(item)
    try {
      const detail = await getKnowledgeArticle(item.slug)
      setSelected(detail.article)
    } catch {
      setSelected(item)
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <BookOpen className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-bold">使用教程</h1>
            <p className="text-sm text-muted-foreground">查看客户端导入、节点配置和常见问题处理指南</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-[320px_1fr]">
        <aside className="space-y-4">
          <div className="rounded-xl border bg-card p-4">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input value={keyword} onChange={(e) => setKeyword(e.target.value)} className="pl-9" placeholder="搜索教程" />
            </div>
            <div className="mt-4 flex flex-wrap gap-2">
              <Button size="sm" variant={category === "all" ? "default" : "outline"} onClick={() => setCategory("all")}>
                全部
              </Button>
              {categories.map((item) => (
                <Button key={item} size="sm" variant={category === item ? "default" : "outline"} onClick={() => setCategory(item)}>
                  {item}
                </Button>
              ))}
            </div>
          </div>

          <div className="rounded-xl border bg-card overflow-hidden">
            {filtered.length === 0 ? (
              <div className="px-4 py-10 text-center text-sm text-muted-foreground">暂无教程</div>
            ) : filtered.map((item) => {
              const active = selected?.slug === item.slug
              return (
                <button
                  key={item.slug}
                  type="button"
                  onClick={() => openArticle(item)}
                  className={`flex w-full items-center gap-3 border-b px-4 py-3 text-left transition last:border-b-0 ${
                    active ? "bg-primary/10 text-primary" : "hover:bg-accent/60"
                  }`}
                >
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium">{item.title}</div>
                    <div className="mt-1 truncate text-xs text-muted-foreground">{item.summary || item.category}</div>
                  </div>
                  <ChevronRight className="h-4 w-4 flex-shrink-0 opacity-60" />
                </button>
              )
            })}
          </div>
        </aside>

        <article className="min-h-[520px] rounded-xl border bg-card">
          {selected ? (
            <div className="p-5 sm:p-7">
              <div className="mb-5 flex flex-wrap items-center gap-2">
                <span className="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary">{selected.category || "通用教程"}</span>
                <span className="text-xs text-muted-foreground">
                  {selected.updated_at ? new Date(selected.updated_at).toLocaleString("zh-CN") : ""}
                </span>
              </div>
              <h2 className="text-2xl font-bold tracking-tight">{selected.title}</h2>
              {selected.summary && <p className="mt-3 text-sm text-muted-foreground">{selected.summary}</p>}
              <div className="mt-6 rounded-lg bg-muted/35 p-5">
                <MarkdownRenderer content={selected.content || ""} />
              </div>
            </div>
          ) : (
            <div className="flex h-full min-h-[520px] flex-col items-center justify-center p-8 text-center text-muted-foreground">
              <BookOpen className="mb-3 h-10 w-10" />
              <p>暂无可用教程</p>
            </div>
          )}
        </article>
      </div>
    </div>
  )
}
