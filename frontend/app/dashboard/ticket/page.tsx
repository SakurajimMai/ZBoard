"use client"

import { useEffect, useState } from "react"
import { Plus, MessageSquare, Send } from "lucide-react"
import { Button } from "@/components/ui/button"
import { getTickets, createTicket, getTicketDetail, replyTicket } from "@/lib/api"

export default function TicketPage() {
  const [tickets, setTickets] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [selectedTicket, setSelectedTicket] = useState<any>(null)
  const [messages, setMessages] = useState<any[]>([])
  const [newSubject, setNewSubject] = useState("")
  const [newCategory, setNewCategory] = useState("general")
  const [newContent, setNewContent] = useState("")
  const [replyContent, setReplyContent] = useState("")
  const [submitting, setSubmitting] = useState(false)

  const load = () => {
    setLoading(true)
    getTickets()
      .then((res) => setTickets(res.items || []))
      .catch(console.error)
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [])

  const handleCreate = async () => {
    if (!newSubject.trim() || !newContent.trim()) return
    setSubmitting(true)
    try {
      await createTicket(newSubject, newCategory, newContent)
      setShowForm(false)
      setNewSubject("")
      setNewContent("")
      load()
    } catch (err: any) {
      alert(err.message || "提交失败")
    } finally {
      setSubmitting(false)
    }
  }

  const openTicket = async (ticketNo: string) => {
    try {
      const res = await getTicketDetail(ticketNo)
      setSelectedTicket(res.ticket)
      setMessages(res.messages || [])
    } catch (err: any) {
      alert(err.message || "加载失败")
    }
  }

  const handleReply = async () => {
    if (!replyContent.trim() || !selectedTicket) return
    setSubmitting(true)
    try {
      await replyTicket(selectedTicket.ticket_no, replyContent)
      setReplyContent("")
      openTicket(selectedTicket.ticket_no)
    } catch (err: any) {
      alert(err.message || "回复失败")
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  if (selectedTicket) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <button onClick={() => setSelectedTicket(null)} className="text-sm text-primary hover:underline">← 返回列表</button>
          <span className="text-xs text-muted-foreground font-mono">{selectedTicket.ticket_no}</span>
          <span className={`text-xs rounded-full px-2 py-0.5 ${
            selectedTicket.status === "open" ? "bg-yellow-100 text-yellow-700" :
            selectedTicket.status === "replied" ? "bg-blue-100 text-blue-700" :
            "bg-gray-100 text-gray-600"
          }`}>{selectedTicket.status === "open" ? "待回复" : selectedTicket.status === "replied" ? "已回复" : "已关闭"}</span>
        </div>

        <h1 className="text-xl font-bold">{selectedTicket.subject}</h1>

        <div className="space-y-4">
          {messages.map((msg: any) => (
            <div key={msg.id} className={`rounded-xl p-4 ${
              msg.sender_type === "admin" ? "bg-primary/5 border border-primary/20" : "bg-muted"
            }`}>
              <div className="flex items-center gap-2 mb-2">
                <span className={`text-xs font-medium ${msg.sender_type === "admin" ? "text-primary" : "text-foreground"}`}>
                  {msg.sender_type === "admin" ? "客服" : "我"}
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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">工单支持</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {tickets.length > 0 ? `共 ${tickets.length} 个工单` : "遇到问题？提交工单获取帮助"}
          </p>
        </div>
        <Button onClick={() => setShowForm(!showForm)}>
          <Plus className="w-4 h-4 mr-1" /> 新建工单
        </Button>
      </div>

      {showForm && (
        <div className="rounded-xl border p-5 space-y-4 bg-card">
          <h3 className="font-medium">新建工单</h3>
          <div>
            <label className="text-sm text-muted-foreground">标题</label>
            <input
              value={newSubject}
              onChange={(e) => setNewSubject(e.target.value)}
              className="mt-1 w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="简述您的问题"
            />
          </div>
          <div>
            <label className="text-sm text-muted-foreground">分类</label>
            <select
              value={newCategory}
              onChange={(e) => setNewCategory(e.target.value)}
              className="mt-1 w-full rounded-lg border bg-background px-3 py-2 text-sm"
            >
              <option value="general">一般问题</option>
              <option value="connection">连接问题</option>
              <option value="billing">账单问题</option>
              <option value="account">账户问题</option>
            </select>
          </div>
          <div>
            <label className="text-sm text-muted-foreground">详细描述</label>
            <textarea
              value={newContent}
              onChange={(e) => setNewContent(e.target.value)}
              rows={4}
              className="mt-1 w-full rounded-lg border bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="请详细描述您遇到的问题..."
            />
          </div>
          <div className="flex gap-2">
            <Button onClick={handleCreate} disabled={submitting}>
              {submitting ? "提交中..." : "提交工单"}
            </Button>
            <Button variant="outline" onClick={() => setShowForm(false)}>取消</Button>
          </div>
        </div>
      )}

      {tickets.length === 0 && !showForm ? (
        <div className="flex flex-col items-center py-16 text-center">
          <MessageSquare className="w-12 h-12 text-muted-foreground mb-4" />
          <p className="text-muted-foreground">暂无工单</p>
          <p className="text-sm text-muted-foreground mt-1">点击"新建工单"提交您的问题</p>
        </div>
      ) : (
        <div className="space-y-3">
          {tickets.map((t: any) => (
            <div
              key={t.id}
              onClick={() => openTicket(t.ticket_no)}
              className="rounded-xl border bg-card p-4 hover:border-primary/30 transition-colors cursor-pointer"
            >
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
                {t.category} · {new Date(t.created_at).toLocaleDateString("zh-CN")}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
