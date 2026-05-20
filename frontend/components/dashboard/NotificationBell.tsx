"use client"

import { useEffect, useState } from "react"
import { Bell, Check } from "lucide-react"
import { Button } from "@/components/ui/button"
import { getNotifications, markNotificationRead, markAllNotificationsRead, getUnreadCount } from "@/lib/api"

export default function NotificationBell() {
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<any[]>([])
  const [unread, setUnread] = useState(0)

  const load = () => {
    getUnreadCount().then((res) => setUnread(res.unread)).catch(() => {})
  }

  useEffect(() => {
    load()
    const interval = setInterval(load, 30000) // poll every 30s
    return () => clearInterval(interval)
  }, [])

  const handleOpen = async () => {
    if (!open) {
      try {
        const res = await getNotifications()
        setItems(res.items || [])
        setUnread(res.unread)
      } catch {}
    }
    setOpen(!open)
  }

  const handleRead = async (id: number) => {
    await markNotificationRead(id).catch(() => {})
    setItems(items.map(n => n.id === id ? { ...n, is_read: 1 } : n))
    setUnread(Math.max(0, unread - 1))
  }

  const handleReadAll = async () => {
    await markAllNotificationsRead().catch(() => {})
    setItems(items.map(n => ({ ...n, is_read: 1 })))
    setUnread(0)
  }

  return (
    <div className="relative">
      <button
        onClick={handleOpen}
        className="relative p-2 rounded-lg hover:bg-accent transition-colors"
        aria-label="通知"
      >
        <Bell className="w-5 h-5" />
        {unread > 0 && (
          <span className="absolute -top-0.5 -right-0.5 w-4 h-4 bg-red-500 text-white text-[10px] font-bold rounded-full flex items-center justify-center">
            {unread > 9 ? "9+" : unread}
          </span>
        )}
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-full mt-2 w-80 max-h-96 overflow-y-auto rounded-xl border bg-card shadow-lg z-50">
            <div className="flex items-center justify-between px-4 py-3 border-b">
              <span className="font-medium text-sm">通知</span>
              {unread > 0 && (
                <button onClick={handleReadAll} className="text-xs text-primary hover:underline">
                  全部已读
                </button>
              )}
            </div>
            {items.length === 0 ? (
              <div className="p-6 text-center text-sm text-muted-foreground">暂无通知</div>
            ) : (
              <div className="divide-y">
                {items.map((n: any) => (
                  <div
                    key={n.id}
                    className={`px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors ${
                      n.is_read === 0 ? "bg-primary/5" : ""
                    }`}
                    onClick={() => {
                      if (n.is_read === 0) handleRead(n.id)
                      if (n.link) window.location.href = n.link
                    }}
                  >
                    <div className="flex items-start gap-2">
                      {n.is_read === 0 && (
                        <div className="w-2 h-2 rounded-full bg-primary mt-1.5 flex-shrink-0" />
                      )}
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate">{n.title}</p>
                        <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">{n.content}</p>
                        <p className="text-xs text-muted-foreground mt-1">
                          {new Date(n.created_at).toLocaleString("zh-CN")}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}
