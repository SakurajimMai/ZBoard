"use client"

import { useEffect, useMemo, useState } from "react"
import { Bell } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { getAnnouncements } from "@/lib/api"

export default function AnnouncementPopup() {
  const [items, setItems] = useState<any[]>([])
  const [open, setOpen] = useState(false)

  useEffect(() => {
    getAnnouncements()
      .then((res) => {
        const popups = (res.items || []).filter((item: any) => item.popup)
        setItems(popups)
        const first = popups[0]
        if (!first) return
        const key = `zboard_announcement_seen_${first.id}`
        if (localStorage.getItem(key) !== "1") {
          setOpen(true)
        }
      })
      .catch(() => {})
  }, [])

  const current = useMemo(() => items[0], [items])
  if (!current) return null

  const close = () => {
    localStorage.setItem(`zboard_announcement_seen_${current.id}`, "1")
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) close(); else setOpen(true) }}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Bell className="w-5 h-5 text-primary" /> {current.title}
          </DialogTitle>
        </DialogHeader>
        <div className="whitespace-pre-wrap text-sm leading-6 text-muted-foreground">
          {current.content}
        </div>
        <div className="flex justify-end">
          <Button onClick={close}>知道了</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
