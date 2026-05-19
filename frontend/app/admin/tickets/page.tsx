"use client"

import { MessageSquare, CheckCheck } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useState } from "react"

const tickets = [
  { id: "TK-2024-0088", user: "alice@gmail.com", subject: "节点连接速度异常", status: "open", priority: "high", date: "2024-05-16", replies: 2 },
  { id: "TK-2024-0087", user: "bob@qq.com", subject: "忘记密码无法登录", status: "open", priority: "medium", date: "2024-05-17", replies: 1 },
  { id: "TK-2024-0086", user: "grace@gmail.com", subject: "能否增加台湾 IPLC 节点", status: "open", priority: "low", date: "2024-05-17", replies: 0 },
  { id: "TK-2024-0085", user: "henry@yahoo.com", subject: "订阅链接无法在 Shadowrocket 导入", status: "resolved", priority: "medium", date: "2024-05-14", replies: 5 },
  { id: "TK-2024-0084", user: "charlie@163.com", subject: "申请退款", status: "resolved", priority: "high", date: "2024-05-12", replies: 3 },
]

const priorityMap: Record<string, string> = {
  high: "text-red-400 bg-red-500/10",
  medium: "text-yellow-400 bg-yellow-500/10",
  low: "text-muted-foreground bg-secondary",
}

const priorityLabel: Record<string, string> = {
  high: "紧急",
  medium: "普通",
  low: "低优先",
}

export default function AdminTicketsPage() {
  const [filter, setFilter] = useState<"all" | "open" | "resolved">("all")

  const filtered = tickets.filter((t) => filter === "all" || t.status === filter)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">工单管理</h1>
          <p className="text-sm text-muted-foreground mt-1">
            <span className="text-primary">{tickets.filter((t) => t.status === "open").length} 待处理</span>
            {" · "}{tickets.length} 共计
          </p>
        </div>
      </div>

      <div className="flex gap-2">
        {(["all", "open", "resolved"] as const).map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-4 py-1.5 rounded-lg text-sm transition-colors ${
              filter === f ? "bg-primary text-primary-foreground" : "bg-secondary text-muted-foreground hover:text-foreground border border-border"
            }`}
          >
            {{ all: "全部", open: "待处理", resolved: "已解决" }[f]}
          </button>
        ))}
      </div>

      <div className="space-y-3">
        {filtered.map((t) => (
          <div key={t.id} className="rounded-xl border border-border bg-card p-4 hover:border-primary/30 transition-colors cursor-pointer">
            <div className="flex items-start justify-between gap-4">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-mono text-xs text-muted-foreground">{t.id}</span>
                  <span className={`text-xs rounded px-2 py-0.5 ${priorityMap[t.priority]}`}>
                    {priorityLabel[t.priority]}
                  </span>
                  <span className={`text-xs rounded-full px-2.5 py-0.5 ${
                    t.status === "open" ? "bg-primary/20 text-primary" : "bg-secondary text-muted-foreground"
                  }`}>
                    {t.status === "open" ? "待处理" : "已解决"}
                  </span>
                </div>
                <p className="font-medium text-foreground">{t.subject}</p>
                <p className="text-sm text-muted-foreground mt-0.5">{t.user} · {t.date}</p>
              </div>
              <div className="flex items-center gap-3 flex-shrink-0">
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
                  <MessageSquare className="w-3.5 h-3.5" />
                  {t.replies}
                </span>
                {t.status === "open" && (
                  <Button size="sm" className="gap-1.5 bg-primary text-primary-foreground hover:bg-primary/90 h-7 text-xs">
                    <CheckCheck className="w-3.5 h-3.5" />
                    回复
                  </Button>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
