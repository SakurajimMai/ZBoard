"use client"

import { useEffect, useState } from "react"
import { Users, Server, DollarSign, TrendingUp, CheckCircle2 } from "lucide-react"
import { adminGetOverview, adminGetUsers } from "@/lib/api"

export default function AdminOverview() {
  const [stats, setStats] = useState({ users: 0, nodes: 0, orders: 0, revenue: "0" })
  const [users, setUsers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      adminGetUsers({ page: 1, pageSize: 5 }),
      adminGetOverview(),
    ])
      .then(([usersRes, overview]) => {
        const allUsers = usersRes.items || []

        setStats({
          users: overview.users,
          nodes: overview.active_nodes,
          orders: overview.paid_orders,
          revenue: overview.revenue,
        })
        setUsers(allUsers.slice(0, 5))
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  const today = new Date().toLocaleDateString("zh-CN", { year: "numeric", month: "long", day: "numeric" })

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

      {/* KPI cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard icon={Users} label="总用户数" value={String(stats.users)} />
        <StatCard icon={Server} label="在线节点" value={String(stats.nodes)} />
        <StatCard icon={DollarSign} label="总收入" value={`¥${stats.revenue}`} />
        <StatCard icon={TrendingUp} label="已付订单" value={String(stats.orders)} />
      </div>

      {/* Recent users */}
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
