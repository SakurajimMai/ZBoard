"use client"

import { useState } from "react"
import { Search, UserPlus, MoreHorizontal, Ban, Edit, RefreshCw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

const users = [
  { id: 1, email: "alice@gmail.com", plan: "标准版", traffic: "87/300 GB", expire: "2024-06-18", status: "active", joined: "2024-01-15" },
  { id: 2, email: "bob@qq.com", plan: "入门版", traffic: "45/100 GB", expire: "2024-06-10", status: "active", joined: "2024-02-20" },
  { id: 3, email: "charlie@163.com", plan: "旗舰版", traffic: "320/1024 GB", expire: "2024-12-01", status: "active", joined: "2024-03-05" },
  { id: 4, email: "david@outlook.com", plan: "标准版", traffic: "300/300 GB", expire: "2024-05-01", status: "expired", joined: "2024-02-01" },
  { id: 5, email: "eve@hotmail.com", plan: "入门版", traffic: "12/100 GB", expire: "2024-07-01", status: "active", joined: "2024-05-01" },
  { id: 6, email: "frank@icloud.com", plan: "旗舰版", traffic: "0/1024 GB", expire: "2025-01-15", status: "banned", joined: "2024-04-18" },
  { id: 7, email: "grace@gmail.com", plan: "标准版", traffic: "155/300 GB", expire: "2024-06-28", status: "active", joined: "2024-03-28" },
  { id: 8, email: "henry@yahoo.com", plan: "入门版", traffic: "88/100 GB", expire: "2024-05-30", status: "active", joined: "2024-04-30" },
]

const statusMap: Record<string, { label: string; class: string }> = {
  active: { label: "正常", class: "bg-green-500/15 text-green-400" },
  expired: { label: "已到期", class: "bg-secondary text-muted-foreground" },
  banned: { label: "已封禁", class: "bg-destructive/15 text-destructive-foreground" },
}

export default function AdminUsers() {
  const [search, setSearch] = useState("")
  const [openMenu, setOpenMenu] = useState<number | null>(null)

  const filtered = users.filter(
    (u) => u.email.includes(search) || u.plan.includes(search)
  )

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">用户管理</h1>
          <p className="text-sm text-muted-foreground mt-1">共 {users.length} 位用户</p>
        </div>
        <Button className="gap-2 bg-primary text-primary-foreground hover:bg-primary/90">
          <UserPlus className="w-4 h-4" />
          添加用户
        </Button>
      </div>

      <div className="flex gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="搜索邮箱或套餐..."
            className="pl-9 bg-secondary border-border"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-secondary/50 border-b border-border">
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">用户</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">套餐</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">流量使用</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">到期时间</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">状态</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((u) => (
                <tr key={u.id} className="border-b border-border/50 hover:bg-accent/50 transition-colors">
                  <td className="px-5 py-3">
                    <div className="flex items-center gap-2">
                      <div className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-xs font-bold text-primary flex-shrink-0">
                        {u.email[0].toUpperCase()}
                      </div>
                      <div>
                        <p className="text-foreground">{u.email}</p>
                        <p className="text-xs text-muted-foreground">注册于 {u.joined}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-5 py-3">
                    <span className="text-xs rounded bg-primary/10 text-primary px-2 py-0.5">{u.plan}</span>
                  </td>
                  <td className="px-5 py-3">
                    <p className="text-foreground text-xs">{u.traffic}</p>
                    <div className="mt-1 h-1 w-20 rounded-full bg-secondary overflow-hidden">
                      <div
                        className="h-full rounded-full bg-primary"
                        style={{
                          width: `${Math.round((parseInt(u.traffic) / parseInt(u.traffic.split("/")[1])) * 100)}%`,
                        }}
                      />
                    </div>
                  </td>
                  <td className="px-5 py-3 text-muted-foreground">{u.expire}</td>
                  <td className="px-5 py-3">
                    <span className={`text-xs rounded-full px-2.5 py-1 ${statusMap[u.status].class}`}>
                      {statusMap[u.status].label}
                    </span>
                  </td>
                  <td className="px-5 py-3">
                    <div className="relative">
                      <button
                        onClick={() => setOpenMenu(openMenu === u.id ? null : u.id)}
                        className="p-1.5 rounded hover:bg-accent transition-colors"
                      >
                        <MoreHorizontal className="w-4 h-4 text-muted-foreground" />
                      </button>
                      {openMenu === u.id && (
                        <div className="absolute right-0 top-8 z-10 w-40 rounded-xl border border-border bg-popover shadow-lg py-1">
                          <button className="flex items-center gap-2 px-4 py-2 text-sm text-foreground hover:bg-accent w-full text-left">
                            <Edit className="w-3.5 h-3.5" />
                            编辑用户
                          </button>
                          <button className="flex items-center gap-2 px-4 py-2 text-sm text-foreground hover:bg-accent w-full text-left">
                            <RefreshCw className="w-3.5 h-3.5" />
                            重置流量
                          </button>
                          <button className="flex items-center gap-2 px-4 py-2 text-sm text-destructive-foreground hover:bg-destructive/10 w-full text-left">
                            <Ban className="w-3.5 h-3.5" />
                            封禁账户
                          </button>
                        </div>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="px-5 py-3 border-t border-border text-xs text-muted-foreground">
          显示 {filtered.length} / {users.length} 条记录
        </div>
      </div>
    </div>
  )
}
