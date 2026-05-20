"use client"

import { useEffect, useState } from "react"
import { Globe, Wifi, Signal } from "lucide-react"
import { getMe } from "@/lib/api"

export default function Subscription() {
  const [user, setUser] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    getMe()
      .then((res) => setUser(res.user))
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  const apiBase = process.env.NEXT_PUBLIC_API_URL || (typeof window !== 'undefined' ? window.location.origin.replace(':3001', ':3000') : '')

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">节点状态</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {user?.status === "active"
            ? "您的订阅有效，可使用以下节点。请通过订阅链接导入客户端。"
            : "请先购买套餐以使用节点服务。"}
        </p>
      </div>

      {user?.status !== "active" && (
        <div className="rounded-xl border border-yellow-500/30 bg-yellow-50 p-4">
          <p className="text-sm text-yellow-800">
            您当前没有有效套餐。请前往 <a href="/dashboard/billing" className="underline font-medium">账单页面</a> 购买。
          </p>
        </div>
      )}

      {user?.status === "active" && (
        <div className="rounded-xl border bg-card p-4 space-y-3">
          <h3 className="font-medium flex items-center gap-2">
            <Globe className="w-4 h-4" /> 使用说明
          </h3>
          <div className="text-sm text-muted-foreground space-y-2">
            <p>1. 前往 <a href="/dashboard" className="text-primary underline">控制台</a> 复制订阅链接</p>
            <p>2. 在 Clash Meta / sing-box / V2rayN 中导入订阅</p>
            <p>3. 选择节点并连接</p>
          </div>
          <div className="mt-3 text-xs text-muted-foreground">
            <p>订阅链接格式：</p>
            <ul className="list-disc list-inside mt-1 space-y-1">
              <li>默认 (Base64): <code>?target=base64</code></li>
              <li>Clash Meta: <code>?target=clash</code></li>
              <li>sing-box: <code>?target=sing-box</code></li>
            </ul>
          </div>
        </div>
      )}

      <div className="rounded-xl border bg-card p-4">
        <h3 className="font-medium flex items-center gap-2 mb-3">
          <Wifi className="w-4 h-4" /> 账户信息
        </h3>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-muted-foreground">流量已用</span>
            <p className="font-medium">{((user?.traffic_used || 0) / 1073741824).toFixed(1)} GB / {((user?.traffic_limit || 0) / 1073741824).toFixed(0)} GB</p>
          </div>
          <div>
            <span className="text-muted-foreground">到期时间</span>
            <p className="font-medium">{user?.expired_at ? new Date(user.expired_at).toLocaleDateString("zh-CN") : "无"}</p>
          </div>
        </div>
      </div>
    </div>
  )
}
