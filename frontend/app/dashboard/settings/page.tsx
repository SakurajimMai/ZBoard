"use client"

import { useEffect, useState } from "react"
import { CircleUserRound } from "lucide-react"
import { Input } from "@/components/ui/input"
import { getMe } from "@/lib/api"

export default function SettingsPage() {
  const [user, setUser] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    getMe()
      .then((res) => setUser(res.user))
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div className="text-muted-foreground p-8">加载中...</div>
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">账户设置</h1>
        <p className="text-sm text-muted-foreground mt-1">查看当前登录账户信息。</p>
      </div>

      <div className="rounded-xl border border-border bg-card p-6 space-y-5">
        <h2 className="font-semibold text-foreground">基本信息</h2>
        <div className="flex items-center gap-4">
          <CircleUserRound className="w-12 h-12 text-foreground/80 flex-shrink-0" strokeWidth={1.8} />
          <div className="flex-1 min-w-0">
            <label className="text-sm text-muted-foreground mb-1.5 block">邮箱地址</label>
            <Input value={user?.email || ""} readOnly className="bg-secondary border-border" />
          </div>
        </div>
      </div>
    </div>
  )
}
