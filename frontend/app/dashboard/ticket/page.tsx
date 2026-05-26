"use client"

import { useCallback, useEffect, useState } from "react"
import { Plus, MessageSquare, Send } from "lucide-react"
import { Button } from "@/components/ui/button"
import { getTickets, createTicket, getTicketDetail, replyTicket, getPublicSettings } from "@/lib/api"
import { Captcha, captchaEnabled } from "@/components/captcha"
import { useI18n } from "@/lib/i18n/context"
import { dashboardCopy } from "@/lib/i18n/dashboard"

export default function TicketPage() {
  const { locale } = useI18n()
  const d = dashboardCopy(locale)
  const [tickets, setTickets] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [selectedTicket, setSelectedTicket] = useState<any>(null)
  const [messages, setMessages] = useState<any[]>([])
  const [newSubject, setNewSubject] = useState("")
  const [newCategory, setNewCategory] = useState("general")
  const [newContent, setNewContent] = useState("")
  const [replyContent, setReplyContent] = useState("")
  const [settings, setSettings] = useState<Record<string, string>>({})
  const [captchaToken, setCaptchaToken] = useState("")
  const [captchaKey, setCaptchaKey] = useState(0)
  const [submitting, setSubmitting] = useState(false)

  const load = () => {
    setLoading(true)
    getTickets()
      .then((res) => setTickets(res.items || []))
      .catch(console.error)
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
    getPublicSettings()
      .then((res) => setSettings(res.settings || {}))
      .catch(() => {})
  }, [])

  const needCaptcha = captchaEnabled(settings, "ticket")
  const resetCaptcha = useCallback(() => {
    setCaptchaToken("")
    setCaptchaKey((v) => v + 1)
  }, [])

  const handleCreate = async () => {
    if (!newSubject.trim() || !newContent.trim()) return
    if (needCaptcha && !captchaToken) {
      alert(d.ticket.captchaRequired)
      return
    }
    setSubmitting(true)
    try {
      await createTicket(newSubject, newCategory, newContent, captchaToken)
      setShowForm(false)
      setNewSubject("")
      setNewContent("")
      load()
    } catch (err: any) {
      alert(err.message || d.ticket.submitFailed)
    } finally {
      if (needCaptcha) resetCaptcha()
      setSubmitting(false)
    }
  }

  const openTicket = async (ticketNo: string) => {
    try {
      const res = await getTicketDetail(ticketNo)
      setSelectedTicket(res.ticket)
      setMessages(res.messages || [])
    } catch (err: any) {
      alert(err.message || d.ticket.loadFailed)
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
      alert(err.message || d.ticket.replyFailed)
    } finally {
      setSubmitting(false)
    }
  }

  const ticketStatus = (status: string) => {
    if (status === "open") return d.ticket.statusOpen
    if (status === "replied") return d.ticket.statusReplied
    return d.ticket.statusClosed
  }

  if (loading) return <div className="text-muted-foreground p-8">{d.common.loading}</div>

  if (selectedTicket) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <button onClick={() => setSelectedTicket(null)} className="text-sm text-primary hover:underline">{d.ticket.backToList}</button>
          <span className="text-xs text-muted-foreground font-mono">{selectedTicket.ticket_no}</span>
          <span className={`text-xs rounded-full px-2 py-0.5 ${
            selectedTicket.status === "open" ? "bg-yellow-100 text-yellow-700" :
            selectedTicket.status === "replied" ? "bg-blue-100 text-blue-700" :
            "bg-gray-100 text-gray-600"
          }`}>{ticketStatus(selectedTicket.status)}</span>
        </div>

        <h1 className="text-xl font-bold">{selectedTicket.subject}</h1>

        <div className="space-y-4">
          {messages.map((msg: any) => (
            <div key={msg.id} className={`rounded-xl p-4 ${
              msg.sender_type === "admin" ? "bg-primary/5 border border-primary/20" : "bg-muted"
            }`}>
              <div className="flex items-center gap-2 mb-2">
                <span className={`text-xs font-medium ${msg.sender_type === "admin" ? "text-primary" : "text-foreground"}`}>
                  {msg.sender_type === "admin" ? d.ticket.admin : d.ticket.me}
                </span>
                <span className="text-xs text-muted-foreground">
                  {new Date(msg.created_at).toLocaleString(locale)}
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
              placeholder={d.ticket.replyPlaceholder}
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
          <h1 className="text-2xl font-bold">{d.ticket.title}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {tickets.length > 0 ? d.ticket.ticketCount.replace("{count}", String(tickets.length)) : d.ticket.noTicketDesc}
          </p>
        </div>
        <Button onClick={() => setShowForm(!showForm)}>
          <Plus className="w-4 h-4 mr-1" /> {d.ticket.newTicket}
        </Button>
      </div>

      {showForm && (
        <div className="rounded-xl border p-5 space-y-4 bg-card">
          <h3 className="font-medium">{d.ticket.newTicket}</h3>
          <div>
            <label className="text-sm text-muted-foreground">{d.ticket.subject}</label>
            <input
              value={newSubject}
              onChange={(e) => setNewSubject(e.target.value)}
              className="mt-1 w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder={d.ticket.subjectPlaceholder}
            />
          </div>
          <div>
            <label className="text-sm text-muted-foreground">{d.ticket.category}</label>
            <select
              value={newCategory}
              onChange={(e) => setNewCategory(e.target.value)}
              className="mt-1 w-full rounded-lg border bg-background px-3 py-2 text-sm"
            >
              <option value="general">{d.ticket.categoryGeneral}</option>
              <option value="connection">{d.ticket.categoryConnection}</option>
              <option value="billing">{d.ticket.categoryBilling}</option>
              <option value="account">{d.ticket.categoryAccount}</option>
            </select>
          </div>
          <div>
            <label className="text-sm text-muted-foreground">{d.ticket.description}</label>
            <textarea
              value={newContent}
              onChange={(e) => setNewContent(e.target.value)}
              rows={4}
              className="mt-1 w-full rounded-lg border bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder={d.ticket.descPlaceholder}
            />
          </div>
          {needCaptcha && (
            <Captcha
              key={captchaKey}
              provider={settings.captcha_provider as any}
              siteKey={settings.captcha_site_key || ""}
              mode={(settings.turnstile_mode as any) || "managed"}
              onToken={setCaptchaToken}
              onError={(msg) => alert(msg)}
            />
          )}
          <div className="flex gap-2">
            <Button onClick={handleCreate} disabled={submitting}>
              {submitting ? d.ticket.submitting : d.ticket.submitTicket}
            </Button>
            <Button variant="outline" onClick={() => setShowForm(false)}>{d.common.cancel}</Button>
          </div>
        </div>
      )}

      {tickets.length === 0 && !showForm ? (
        <div className="flex flex-col items-center py-16 text-center">
          <MessageSquare className="w-12 h-12 text-muted-foreground mb-4" />
          <p className="text-muted-foreground">{d.ticket.noTickets}</p>
          <p className="text-sm text-muted-foreground mt-1">{d.ticket.noTicketsHint}</p>
        </div>
      ) : (
        <div className="space-y-3">
          {tickets.map((tk: any) => (
            <div
              key={tk.id}
              onClick={() => openTicket(tk.ticket_no)}
              className="rounded-xl border bg-card p-4 hover:border-primary/30 transition-colors cursor-pointer"
            >
              <div className="flex items-center gap-2 mb-1">
                <span className="font-mono text-xs text-muted-foreground">{tk.ticket_no}</span>
                <span className={`text-xs rounded-full px-2 py-0.5 ${
                  tk.status === "open" ? "bg-yellow-100 text-yellow-700" :
                  tk.status === "replied" ? "bg-blue-100 text-blue-700" :
                  "bg-gray-100 text-gray-600"
                }`}>
                  {ticketStatus(tk.status)}
                </span>
              </div>
              <p className="font-medium">{tk.subject}</p>
              <p className="text-xs text-muted-foreground mt-1">
                {tk.category} · {new Date(tk.created_at).toLocaleDateString(locale)}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
