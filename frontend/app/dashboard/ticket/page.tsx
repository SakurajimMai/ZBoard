"use client"

import { useState } from "react"
import { Plus, MessageSquare } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

const tickets = [
  { id: "TK-2024-0088", subject: "节点连接速度异常", status: "open", date: "2024-05-16", reply: 2 },
  { id: "TK-2024-0071", subject: "订阅链接无法导入 Clash", status: "resolved", date: "2024-05-10", reply: 5 },
  { id: "TK-2024-0055", subject: "申请退款", status: "resolved", date: "2024-04-28", reply: 3 },
]

export default function TicketPage() {
  const [showForm, setShowForm] = useState(false)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">工单支持</h1>
          <p className="text-sm text-muted-foreground mt-1">遇到问题？提交工单，我们的技术团队将在 2 小时内响应。</p>
        </div>
        <Button className="gap-2 bg-primary text-primary-foreground" onClick={() => setShowForm(!showForm)}>
          <Plus className="w-4 h-4" />
          新建工单
        </Button>
      </div>

      {showForm && (
        <div className="rounded-xl border border-primary/30 bg-primary/5 p-6 space-y-4">
          <h2 className="font-semibold text-foreground">新建工单</h2>
          <div className="space-y-3">
            <div>
              <label className="text-sm text-muted-foreground mb-1 block">标题</label>
              <Input placeholder="简述您的问题..." className="bg-secondary border-border" />
            </div>
            <div>
              <label className="text-sm text-muted-foreground mb-1 block">分类</label>
              <select className="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground">
                <option>连接问题</option>
                <option>账单问题</option>
                <option>账户问题</option>
                <option>其他</option>
              </select>
            </div>
            <div>
              <label className="text-sm text-muted-foreground mb-1 block">详情描述</label>
              <textarea
                rows={4}
                placeholder="请详细描述您遇到的问题，包括节点名称、客户端版本、错误信息等..."
                className="w-full rounded-lg border border-border bg-secondary px-3 py-2 text-sm text-foreground resize-none focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
            <div className="flex gap-3">
              <Button className="bg-primary text-primary-foreground hover:bg-primary/90">提交工单</Button>
              <Button variant="outline" onClick={() => setShowForm(false)}>取消</Button>
            </div>
          </div>
        </div>
      )}

      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-secondary/50 border-b border-border">
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">工单号</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">标题</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">回复</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">创建时间</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">状态</th>
              </tr>
            </thead>
            <tbody>
              {tickets.map((t) => (
                <tr key={t.id} className="border-b border-border/50 hover:bg-accent/50 transition-colors cursor-pointer">
                  <td className="px-5 py-3 font-mono text-xs text-muted-foreground">{t.id}</td>
                  <td className="px-5 py-3 text-foreground">{t.subject}</td>
                  <td className="px-5 py-3">
                    <span className="flex items-center gap-1 text-muted-foreground">
                      <MessageSquare className="w-3.5 h-3.5" />
                      {t.reply}
                    </span>
                  </td>
                  <td className="px-5 py-3 text-muted-foreground">{t.date}</td>
                  <td className="px-5 py-3">
                    <span className={`text-xs rounded-full px-2.5 py-1 ${
                      t.status === "open"
                        ? "bg-primary/20 text-primary"
                        : "bg-secondary text-muted-foreground"
                    }`}>
                      {t.status === "open" ? "处理中" : "已解决"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
