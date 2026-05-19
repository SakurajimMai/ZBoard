"use client"

import { Copy, RefreshCw, AlertCircle, TrendingUp, Wifi, Clock } from "lucide-react"
import { Button } from "@/components/ui/button"
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts"

const trafficData = [
  { day: "5/12", used: 2.1 },
  { day: "5/13", used: 3.5 },
  { day: "5/14", used: 1.8 },
  { day: "5/15", used: 5.2 },
  { day: "5/16", used: 4.0 },
  { day: "5/17", used: 6.1 },
  { day: "5/18", used: 3.3 },
]

const subscriptionUrl = "https://sub.zboard.io/api/v1/client/subscribe?token=abc123xyz"

export default function Overview() {
  const handleCopy = () => {
    navigator.clipboard.writeText(subscriptionUrl)
  }

  const usedGB = 87.4
  const totalGB = 300
  const usedPct = Math.round((usedGB / totalGB) * 100)

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl sm:text-3xl font-bold text-foreground">控制台</h1>
        <p className="text-sm text-muted-foreground mt-1">欢迎回来，查看您的账户状态和使用情况。</p>
      </div>

      {/* Alert */}
      <div className="flex items-start gap-3 rounded-2xl border border-yellow-500/30 bg-yellow-50 px-4 py-4 card-shadow">
        <div className="w-8 h-8 rounded-xl bg-yellow-100 flex items-center justify-center flex-shrink-0">
          <AlertCircle className="w-4 h-4 text-yellow-600" />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm text-yellow-800 font-medium">套餐即将到期</p>
          <p className="text-sm text-yellow-700 mt-0.5">
            您的套餐将于 <strong>2024 年 6 月 18 日</strong> 到期，请及时续费以避免服务中断。
          </p>
          <Button size="sm" variant="outline" className="mt-3 border-yellow-300 text-yellow-700 hover:bg-yellow-100">
            立即续费
          </Button>
        </div>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {[
          {
            label: "剩余流量",
            value: `${(totalGB - usedGB).toFixed(1)} GB`,
            sub: `已用 ${usedGB} GB / ${totalGB} GB`,
            icon: Wifi,
            pct: usedPct,
          },
          {
            label: "套餐有效期",
            value: "31 天",
            sub: "到期：2024-06-18",
            icon: Clock,
          },
          {
            label: "本月使用",
            value: `${usedGB} GB`,
            sub: "较上月 +12%",
            icon: TrendingUp,
          },
        ].map((s) => {
          const Icon = s.icon
          return (
            <div key={s.label} className="rounded-2xl border border-border/50 bg-card p-5 sm:p-6 card-shadow">
              <div className="flex items-center justify-between mb-4">
                <span className="text-sm font-medium text-muted-foreground">{s.label}</span>
                <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                  <Icon className="w-5 h-5 text-primary" />
                </div>
              </div>
              <div className="text-2xl sm:text-3xl font-bold text-foreground mb-1">{s.value}</div>
              <div className="text-xs text-muted-foreground">{s.sub}</div>
              {"pct" in s && (
                <div className="mt-4">
                  <div className="h-2 rounded-full bg-muted overflow-hidden">
                    <div
                      className="h-full rounded-full bg-primary transition-all"
                      style={{ width: `${s.pct}%` }}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground mt-2">{s.pct}% 已使用</p>
                </div>
              )}
            </div>
          )
        })}
      </div>

      {/* Subscription URL */}
      <div className="rounded-2xl border border-border/50 bg-card p-5 sm:p-6 card-shadow">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4">
          <h2 className="font-semibold text-lg text-foreground">订阅链接</h2>
          <Button variant="ghost" size="sm" className="gap-2 text-muted-foreground hover:text-primary w-fit">
            <RefreshCw className="w-4 h-4" />
            重置链接
          </Button>
        </div>
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 rounded-xl bg-muted border border-border px-4 py-3">
          <code className="flex-1 text-xs sm:text-sm font-mono text-muted-foreground truncate">{subscriptionUrl}</code>
          <Button size="sm" className="gap-2 btn-gradient text-primary-foreground flex-shrink-0" onClick={handleCopy}>
            <Copy className="w-4 h-4" />
            复制链接
          </Button>
        </div>
        <p className="text-xs text-muted-foreground mt-3">
          请勿将订阅链接分享给他人，否则可能导致账户被封禁。
        </p>
      </div>

      {/* Traffic chart */}
      <div className="rounded-2xl border border-border/50 bg-card p-5 sm:p-6 card-shadow">
        <h2 className="font-semibold text-lg text-foreground mb-5">最近 7 天流量使用</h2>
        <ResponsiveContainer width="100%" height={200}>
          <AreaChart data={trafficData}>
            <defs>
              <linearGradient id="trafficGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--color-primary)" stopOpacity={0.2} />
                <stop offset="95%" stopColor="var(--color-primary)" stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis dataKey="day" tick={{ fontSize: 12, fill: "var(--color-muted-foreground)" }} axisLine={false} tickLine={false} />
            <YAxis tick={{ fontSize: 12, fill: "var(--color-muted-foreground)" }} axisLine={false} tickLine={false} unit=" GB" width={50} />
            <Tooltip
              contentStyle={{
                background: "var(--color-card)",
                border: "1px solid var(--color-border)",
                borderRadius: "12px",
                fontSize: "12px",
                color: "var(--color-foreground)",
                boxShadow: "0 4px 12px rgba(0,0,0,0.1)",
              }}
              formatter={(v: number) => [`${v} GB`, "使用量"]}
            />
            <Area
              type="monotone"
              dataKey="used"
              stroke="var(--color-primary)"
              strokeWidth={2}
              fill="url(#trafficGrad)"
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
