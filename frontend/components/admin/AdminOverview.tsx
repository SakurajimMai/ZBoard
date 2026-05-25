"use client"

import { useEffect, useMemo, useState } from "react"
import { CheckCircle2, DollarSign, Server, TrendingUp, Users } from "lucide-react"
import { adminGetOverview, adminGetUsers } from "@/lib/api"

type RevenuePoint = { month: string; label: string; revenue: number }
type TrafficPoint = { day: string; label: string; total: number; tb: number }

export default function AdminOverview() {
  const [stats, setStats] = useState({
    users: 0,
    nodes: 0,
    orders: 0,
    revenue: "0.00",
    revenueTrend: [] as RevenuePoint[],
    trafficTrend: [] as TrafficPoint[],
  })
  const [users, setUsers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      adminGetUsers({ page: 1, pageSize: 5 }),
      adminGetOverview(),
    ])
      .then(([usersRes, overview]) => {
        setStats({
          users: overview.users,
          nodes: overview.active_nodes,
          orders: overview.paid_orders,
          revenue: overview.revenue,
          revenueTrend: overview.revenue_trend || [],
          trafficTrend: overview.traffic_trend || [],
        })
        setUsers((usersRes.items || []).slice(0, 5))
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  const today = useMemo(
    () => new Date().toLocaleDateString("zh-CN", { year: "numeric", month: "long", day: "numeric" }),
    [],
  )

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold text-foreground">数据概览</h1>
          <p className="text-sm text-muted-foreground mt-1">今日 · {today}</p>
        </div>
        <div className="flex items-center gap-2 text-sm text-green-600 bg-green-50 px-3 py-2 rounded-xl">
          <CheckCircle2 className="w-4 h-4" />
          <span>系统运行正常</span>
        </div>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard icon={Users} label="总用户数" value={String(stats.users)} />
        <StatCard icon={Server} label="在线节点" value={String(stats.nodes)} />
        <StatCard icon={DollarSign} label="总收入" value={`¥${stats.revenue}`} />
        <StatCard icon={TrendingUp} label="已付订单" value={String(stats.orders)} />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-5">
        <RevenueBarChart data={stats.revenueTrend} />
        <TrafficAreaChart data={stats.trafficTrend} />
      </div>

      <div className="rounded-2xl border bg-card overflow-hidden">
        <div className="px-5 py-4 border-b flex items-center justify-between">
          <h2 className="font-semibold text-lg">最新用户</h2>
          <a href="/admin/users" className="text-sm text-primary hover:underline">查看全部</a>
        </div>
        {users.length === 0 ? (
          <div className="p-6 text-muted-foreground text-center">暂无用户</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted/50 border-b">
                  <th className="text-left px-5 py-3 text-muted-foreground font-medium">邮箱</th>
                  <th className="text-left px-5 py-3 text-muted-foreground font-medium hidden sm:table-cell">注册时间</th>
                  <th className="text-left px-5 py-3 text-muted-foreground font-medium">状态</th>
                </tr>
              </thead>
              <tbody>
                {users.map((u: any) => (
                  <tr key={u.id} className="border-b hover:bg-accent/50">
                    <td className="px-5 py-3 font-medium">{u.email}</td>
                    <td className="px-5 py-3 text-muted-foreground hidden sm:table-cell">
                      {new Date(u.created_at).toLocaleDateString("zh-CN")}
                    </td>
                    <td className="px-5 py-3">
                      <span className={`text-xs rounded-full px-2 py-1 font-medium ${
                        u.status === "active" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-600"
                      }`}>
                        {u.status === "active" ? "正常" : "已禁用"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

function StatCard({ icon: Icon, label, value }: { icon: any; label: string; value: string }) {
  return (
    <div className="rounded-2xl border bg-card p-5">
      <div className="flex items-center justify-between mb-3">
        <span className="text-sm text-muted-foreground">{label}</span>
        <div className="w-9 h-9 rounded-xl bg-primary/10 flex items-center justify-center">
          <Icon className="w-4 h-4 text-primary" />
        </div>
      </div>
      <div className="text-2xl font-bold">{value}</div>
    </div>
  )
}

function RevenueBarChart({ data }: { data: RevenuePoint[] }) {
  const max = Math.max(...data.map((d) => d.revenue), 1)
  const ticks = [4, 3, 2, 1, 0].map((i) => (max * i) / 4)

  return (
    <div className="rounded-2xl border bg-card p-5">
      <h2 className="font-semibold text-lg mb-5">近 6 月收入趋势</h2>
      <div className="relative h-64 pl-12">
        <div className="absolute inset-y-0 left-0 flex w-10 flex-col justify-between text-xs text-muted-foreground">
          {ticks.map((tick) => <span key={tick}>{Math.round(tick)}</span>)}
        </div>
        <div className="absolute inset-x-12 inset-y-0 flex flex-col justify-between">
          {ticks.map((tick) => <div key={tick} className="border-t border-dashed border-border" />)}
        </div>
        <div className="relative z-10 flex h-full items-end gap-4">
          {data.map((item) => (
            <div key={item.month} className="group flex min-w-0 flex-1 flex-col items-center gap-2">
              <div className="relative flex h-52 w-full items-end justify-center">
                <div
                  className="w-full max-w-20 rounded-t-md bg-primary transition group-hover:bg-primary/80"
                  style={{ height: `${Math.max((item.revenue / max) * 100, item.revenue > 0 ? 4 : 0)}%` }}
                />
                <div className="pointer-events-none absolute bottom-full mb-2 hidden rounded-lg border bg-background px-3 py-2 text-xs shadow-md group-hover:block">
                  <div className="text-muted-foreground">{item.label}</div>
                  <div className="font-semibold text-primary">收入 · ¥{item.revenue.toFixed(2)}</div>
                </div>
              </div>
              <span className="text-xs text-muted-foreground">{item.label}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function TrafficAreaChart({ data }: { data: TrafficPoint[] }) {
  const max = Math.max(...data.map((d) => d.tb), 1)
  const width = 560
  const height = 220
  const padding = 24
  const points = data.map((d, i) => {
    const x = data.length <= 1 ? padding : padding + (i * (width - padding * 2)) / (data.length - 1)
    const y = height - padding - (d.tb / max) * (height - padding * 2)
    return { ...d, x, y }
  })
  const line = points.map((p) => `${p.x},${p.y}`).join(" ")
  const area = points.length > 0
    ? `${padding},${height - padding} ${line} ${width - padding},${height - padding}`
    : ""
  const ticks = [3, 2, 1, 0].map((i) => (max * i) / 3)

  return (
    <div className="rounded-2xl border bg-card p-5">
      <h2 className="font-semibold text-lg mb-5">近 7 天总流量 (TB)</h2>
      <div className="relative">
        <svg viewBox={`0 0 ${width} ${height}`} className="h-64 w-full overflow-visible">
          {ticks.map((tick) => {
            const y = height - padding - (tick / max) * (height - padding * 2)
            return (
              <g key={tick}>
                <line x1={padding} y1={y} x2={width - padding} y2={y} stroke="hsl(var(--border))" strokeDasharray="4 4" />
                <text x={0} y={y + 4} className="fill-muted-foreground text-[11px]">{tick.toFixed(max >= 10 ? 0 : 1)} TB</text>
              </g>
            )
          })}
          {area && <polygon points={area} fill="hsl(var(--primary) / 0.10)" />}
          {line && <polyline points={line} fill="none" stroke="hsl(var(--primary))" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />}
          {points.map((p) => (
            <g key={p.day} className="group">
              <circle cx={p.x} cy={p.y} r="4" fill="hsl(var(--primary))" />
              <title>{`${p.label}: ${p.tb.toFixed(3)} TB`}</title>
            </g>
          ))}
        </svg>
        <div className="mt-1 flex justify-between pl-8 text-xs text-muted-foreground">
          {points.map((p) => <span key={p.day}>{p.label}</span>)}
        </div>
      </div>
    </div>
  )
}
