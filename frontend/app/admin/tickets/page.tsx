"use client"

import { useCallback, useEffect, useState } from "react"
import { MessageSquare, Send, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { adminGetTickets, adminGetTicketDetail, adminReplyTicket, adminCloseTicket } from "@/lib/api"

export default function AdminTicketsPage() {
  const [tickets, setTickets] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState("all")
  const [selectedTicket, setSelectedTicket] = useState<any>(null)
  const [messages, setMessages] = useState<any[]>([])
  const [replyContent, setReplyContent] = useState("")
  const [submitting, setSubmitting] = useState(false)

  const load = useCallback(() => {
    setLoading(true)
    adminGetTickets(filter)
      .then((res) => setTickets(res.items || []))
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [filter])

  useEffect(() => { load() }, [load])

  const openTicket = async (id: number) => {
    try {
      const res = await adminGetTicketDetail(id)
      setSelectedTicket(res.ticket)
      setMessages(res.messages || [])
    } catch (err: any) {
      toast.error(err.message || "加载失败")
    }
  }

  const handleReply = async () => {
    if (!replyContent.trim() || !selectedTicket) return
    setSubmitting(true)
    try {
      await adminReplyTicket(selectedTicket.id, replyContent)
      setReplyContent("")
      openTicket(selectedTicket.id)
      load()
    } catch (err: any) {
      toast.error(err.message || "回复失败")
    } finally {
      setSubmitting(false)
    }
  }

  const handleClose = async () => {
    if (!selectedTicket) return
    try {
      await adminCloseTicket(selectedTicket.id)
      openTicket(selectedTicket.id)
      load()
    } catch (err: any) {
      toast.error(err.message || "关闭失败")
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  // Detail view
  if (selectedTicket) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <button onClick={() => setSelectedTicket(null)} className="text-sm text-primary hover:underline">← 返回列表</button>
            <span className="text-xs text-muted-foreground font-mono">{selectedTicket.ticket_no}</span>
            <span className={`text-xs rounded-full px-2 py-0.5 ${
              selectedTicket.status === "open" ? "bg-yellow-100 text-yellow-700" :
              selectedTicket.status === "replied" ? "bg-blue-100 text-blue-700" :
              "bg-gray-100 text-gray-600"
            }`}>{selectedTicket.status === "open" ? "待回复" : selectedTicket.status === "replied" ? "已回复" : "已关闭"}</span>
          </div>
          {selectedTicket.status !== "closed" && (
            <Button size="sm" variant="outline" onClick={handleClose} className="text-red-500 border-red-200 hover:bg-red-50">
              <X className="w-3 h-3 mr-1" /> 关闭工单
            </Button>
          )}
        </div>

        <div>
          <h1 className="text-xl font-bold">{selectedTicket.subject}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            用户 ID: {selectedTicket.user_id} · {selectedTicket.category} · 创建于 {new Date(selectedTicket.created_at).toLocaleString("zh-CN")}
          </p>
        </div>

        <div className="space-y-4">
          {messages.map((msg: any) => (
            <div key={msg.id} className={`rounded-xl p-4 ${
              msg.sender_type === "admin" ? "bg-primary/5 border border-primary/20" : "bg-muted"
            }`}>
              <div className="flex items-center gap-2 mb-2">
                <span className={`text-xs font-medium ${msg.sender_type === "admin" ? "text-primary" : "text-foreground"}`}>
                  {msg.sender_type === "admin" ? "客服 (ID:" + msg.sender_id + ")" : "用户 (ID:" + msg.sender_id + ")"}
                </span>
                <span className="text-xs text-muted-foreground">
                  {new Date(msg.created_at).toLocaleString("zh-CN")}
                </span>
              </div>
              <p className="text-sm whitespace-pre-wrap">{msg.content}</p>
            </div>
          ))}
        </div>

        {selectedTicket.status !== "closed" && (
          <div className="flex gap-2">
            <textarea
              value={replyContent}
              onChange={(e) => setReplyContent(e.target.value)}
              placeholder="输入回复内容..."
              rows={3}
              className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-primary"
            />
            <Button onClick={handleReply} disabled={submitting || !replyContent.trim()} className="self-end">
              <Send className="w-4 h-4" />
            </Button>
          </div>
        )}
      </div>
    )
  }

  // List view
  const openCount = tickets.filter(t => t.status === "open").length

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">工单管理</h1>
        <p className="text-sm text-muted-foreground mt-1">
          <span className="text-primary">{openCount} 待处理</span> · 共 {tickets.length} 个工单
        </p>
      </div>

      <div className="flex gap-2">
        {["all", "open", "replied", "closed"].map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-4 py-1.5 rounded-lg text-sm transition-colors ${
              filter === f ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground hover:text-foreground"
            }`}
          >
            {{ all: "全部", open: "待回复", replied: "已回复", closed: "已关闭" }[f]}
          </button>
        ))}
      </div>

      {tickets.length === 0 ? (
        <div className="flex flex-col items-center py-16 text-center">
          <MessageSquare className="w-12 h-12 text-muted-foreground mb-4" />
          <p className="text-muted-foreground">暂无工单</p>
        </div>
      ) : (
        <div className="space-y-3">
          {tickets.map((t: any) => (
            <div
              key={t.id}
              onClick={() => openTicket(t.id)}
              className="rounded-xl border bg-card p-4 hover:border-primary/30 transition-colors cursor-pointer"
            >
              <div className="flex items-center justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-mono text-xs text-muted-foreground">{t.ticket_no}</span>
                    <span className={`text-xs rounded-full px-2 py-0.5 ${
                      t.status === "open" ? "bg-yellow-100 text-yellow-700" :
                      t.status === "replied" ? "bg-blue-100 text-blue-700" :
                      "bg-gray-100 text-gray-600"
                    }`}>
                      {t.status === "open" ? "待回复" : t.status === "replied" ? "已回复" : "已关闭"}
                    </span>
                  </div>
                  <p className="font-medium">{t.subject}</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    用户 {t.user_id} · {t.category} · {new Date(t.created_at).toLocaleDateString("zh-CN")}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
