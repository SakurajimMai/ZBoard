"use client"

import { useEffect, useState } from "react"
import { Copy, RefreshCw, TrendingUp, Wifi, Clock } from "lucide-react"
import { Button } from "@/components/ui/button"
import { getMe, getSubscription, resetSubscriptionToken } from "@/lib/api"

export default function Overview() {
  const [user, setUser] = useState<any>(null)
  const [subToken, setSubToken] = useState("")
  const [loading, setLoading] = useState(true)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    Promise.all([getMe(), getSubscription()])
      .then(([meRes, subRes]) => {
        setUser(meRes.user)
        setSubToken(subRes.token)
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div className="text-muted-foreground p-8">加载中...</div>
  }

  if (!user) {
    return <div className="text-red-500 p-8">无法加载用户信息</div>
  }

  const apiBase = process.env.NEXT_PUBLIC_API_URL || (typeof window !== 'undefined' ? window.location.origin.replace(':3001', ':3000') : '')
  const subscriptionUrl = `${apiBase}/api/sub/${subToken}`
  const usedBytes = user.traffic_used || 0
  const totalBytes = user.traffic_limit || 0
  const usedGB = (usedBytes / 1073741824).toFixed(1)
  const totalGB = (totalBytes / 1073741824).toFixed(0)
  const usedPct = totalBytes > 0 ? Math.round((usedBytes / totalBytes) * 100) : 0

  const handleCopy = () => {
    navigator.clipboard.writeText(subscriptionUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleReset = async () => {
    try {
      const res = await resetSubscriptionToken()
      setSubToken(res.token)
    } catch (e) {
      console.error(e)
    }
  }

  const expireDate = user.expired_at
    ? new Date(user.expired_at).toLocaleDateString("zh-CN")
    : "未订阅"

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl sm:text-3xl font-bold text-foreground">控制台</h1>
        <p className="text-sm text-muted-foreground mt-1">欢迎回来，{user.email}</p>
      </div>

      {/* Subscription URL */}
      <div className="rounded-2xl border bg-card p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="font-medium flex items-center gap-2">
            <Wifi className="w-4 h-4" /> 订阅链接
          </h3>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" onClick={handleCopy}>
              <Copy className="w-3 h-3 mr-1" /> {copied ? "已复制" : "复制"}
            </Button>
            <Button size="sm" variant="outline" onClick={handleReset}>
              <RefreshCw className="w-3 h-3 mr-1" /> 重置
            </Button>
          </div>
        </div>
        <code className="block text-xs bg-muted rounded-lg p-3 break-all">
          {subscriptionUrl}
        </code>
        <p className="text-xs text-muted-foreground">
          支持 Clash Meta / sing-box / V2rayN。添加 ?target=clash 或 ?target=sing-box 切换格式。
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="rounded-2xl border bg-card p-4">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <TrendingUp className="w-4 h-4" /> 已用流量
          </div>
          <p className="text-2xl font-bold mt-1">{usedGB} GB</p>
          <p className="text-xs text-muted-foreground">/ {totalGB} GB ({usedPct}%)</p>
          <div className="mt-2 h-2 rounded-full bg-muted overflow-hidden">
            <div className="h-full bg-primary rounded-full transition-all" style={{ width: `${Math.min(usedPct, 100)}%` }} />
          </div>
        </div>

        <div className="rounded-2xl border bg-card p-4">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Clock className="w-4 h-4" /> 到期时间
          </div>
          <p className="text-2xl font-bold mt-1">{expireDate}</p>
          <p className="text-xs text-muted-foreground">
            {user.status === "active" ? "套餐生效中" : "已过期或未订阅"}
          </p>
        </div>

        <div className="rounded-2xl border bg-card p-4">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Wifi className="w-4 h-4" /> 账户状态
          </div>
          <p className={`text-2xl font-bold mt-1 ${user.status === "active" ? "text-green-600" : "text-red-500"}`}>
            {user.status === "active" ? "正常" : "已禁用"}
          </p>
          <p className="text-xs text-muted-foreground">{user.email}</p>
        </div>
      </div>
    </div>
  )
}
