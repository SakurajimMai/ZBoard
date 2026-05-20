"use client"

import { useEffect, useState } from "react"
import { adminGetUsers } from "@/lib/api"

export default function AdminUsers() {
  const [users, setUsers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    adminGetUsers()
      .then((res) => setUsers(res.items || []))
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">用户管理</h1>
        <p className="text-sm text-muted-foreground mt-1">共 {users.length} 个用户</p>
      </div>

      {users.length === 0 ? (
        <div className="text-center text-muted-foreground py-12">暂无用户</div>
      ) : (
        <div className="rounded-xl border bg-card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted/50 border-b">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ID</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">邮箱</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden md:table-cell">流量</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">到期时间</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">状态</th>
                </tr>
              </thead>
              <tbody>
                {users.map((u: any) => (
                  <tr key={u.id} className="border-b hover:bg-accent/50">
                    <td className="px-4 py-3">{u.id}</td>
                    <td className="px-4 py-3 font-medium">{u.email}</td>
                    <td className="px-4 py-3 hidden md:table-cell text-muted-foreground">
                      {((u.traffic_used || 0) / 1073741824).toFixed(1)} / {((u.traffic_limit || 0) / 1073741824).toFixed(0)} GB
                    </td>
                    <td className="px-4 py-3 hidden lg:table-cell text-muted-foreground">
                      {u.expired_at ? new Date(u.expired_at).toLocaleDateString("zh-CN") : "-"}
                    </td>
                    <td className="px-4 py-3">
                      <span className={`text-xs rounded-full px-2 py-1 ${
                        u.status === "active" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"
                      }`}>
                        {u.status === "active" ? "正常" : "已禁用"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
