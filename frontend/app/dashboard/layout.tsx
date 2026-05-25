"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { getToken, getMe } from "@/lib/api"
import Sidebar from "@/components/dashboard/Sidebar"
import AnnouncementPopup from "@/components/dashboard/AnnouncementPopup"

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const [ready, setReady] = useState(false)
  const [user, setUser] = useState<any>(null)

  useEffect(() => {
    const token = getToken()
    if (!token) {
      router.replace("/login")
      return
    }
    getMe().then((res) => {
      setUser(res.user)
      setReady(true)
    }).catch(() => {
      router.replace("/login")
    })
  }, [router])

  if (!ready) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-muted-foreground">加载中...</p>
      </div>
    )
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar user={user} />
      <main className="flex-1 overflow-y-auto pt-16 lg:pt-0">
        <div className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
          {children}
        </div>
      </main>
      <AnnouncementPopup />
    </div>
  )
}
