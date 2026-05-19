"use client"

import { Users, Server, DollarSign, TrendingUp, ArrowUpRight, ArrowDownRight, Activity, CheckCircle2 } from "lucide-react"
import {
  AreaChart, Area, BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from "recharts"

const revenueData = [
  { month: "12月", revenue: 18200 },
  { month: "1月", revenue: 21500 },
  { month: "2月", revenue: 19800 },
  { month: "3月", revenue: 25600 },
  { month: "4月", revenue: 28100 },
  { month: "5月", revenue: 31450 },
]

const trafficData = [
  { day: "5/12", tb: 4.2 },
  { day: "5/13", tb: 5.8 },
  { day: "5/14", tb: 3.9 },
  { day: "5/15", tb: 7.1 },
  { day: "5/16", tb: 6.5 },
  { day: "5/17", tb: 8.3 },
  { day: "5/18", tb: 5.6 },
]

const recentUsers = [
  { email: "alice@gmail.com", plan: "标准版", joined: "2024-05-18", status: "active" },
  { email: "bob@qq.com", plan: "入门版", joined: "2024-05-17", status: "active" },
  { email: "charlie@163.com", plan: "旗舰版", joined: "2024-05-17", status: "active" },
  { email: "david@outlook.com", plan: "标准版", joined: "2024-05-16", status: "expired" },
  { email: "eve@hotmail.com", plan: "入门版", joined: "2024-05-16", status: "active" },
]

const stats = [
  { label: "总用户数", value: "12,481", change: "+8.2%", up: true, icon: Users },
  { label: "在线节点", value: "198 / 200", change: "2 维护中", up: true, icon: Server },
  { label: "本月收入", value: "¥31,450", change: "+12.0%", up: true, icon: DollarSign },
  { label: "今日流量", value: "5.6 TB", change: "-2.1%", up: false, icon: TrendingUp },
]

const tooltipStyle = {
  background: "var(--color-card)",
  border: "1px solid var(--color-border)",
  borderRadius: "12px",
  fontSize: "12px",
  color: "var(--color-foreground)",
  boxShadow: "0 4px 12px rgba(0,0,0,0.1)",
}

export default function AdminOverview() {
  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold text-foreground">数据概览</h1>
          <p className="text-sm text-muted-foreground mt-1">今日 · 2024年5月18日</p>
        </div>
        <div className="flex items-center gap-2 text-sm text-success bg-success/10 px-3 py-2 rounded-xl">
          <CheckCircle2 className="w-4 h-4" />
          <span>系统运行正常</span>
        </div>
      </div>

      {/* KPI cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((s) => {
          const Icon = s.icon
          return (
            <div key={s.label} className="rounded-2xl border border-border/50 bg-card p-5 sm:p-6 card-shadow">
              <div className="flex items-center justify-between mb-4">
                <span className="text-sm font-medium text-muted-foreground">{s.label}</span>
                <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                  <Icon className="w-5 h-5 text-primary" />
                </div>
              </div>
              <div className="text-xl sm:text-2xl font-bold text-foreground mb-1">{s.value}</div>
              <div className={`flex items-center gap-1 text-xs font-medium ${s.up ? "text-success" : "text-destructive"}`}>
                {s.up ? <ArrowUpRight className="w-3 h-3" /> : <ArrowDownRight className="w-3 h-3" />}
                {s.change}
              </div>
            </div>
          )
        })}
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="rounded-2xl border border-border/50 bg-card p-5 sm:p-6 card-shadow">
          <h2 className="font-semibold text-lg text-foreground mb-5">近 6 月收入趋势</h2>
          <ResponsiveContainer width="100%" height={220}>
            <BarChart data={revenueData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" vertical={false} />
              <XAxis dataKey="month" tick={{ fontSize: 12, fill: "var(--color-muted-foreground)" }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 12, fill: "var(--color-muted-foreground)" }} axisLine={false} tickLine={false} width={50} />
              <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => [`¥${v.toLocaleString()}`, "收入"]} />
              <Bar dataKey="revenue" fill="var(--color-primary)" radius={[6, 6, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="rounded-2xl border border-border/50 bg-card p-5 sm:p-6 card-shadow">
          <h2 className="font-semibold text-lg text-foreground mb-5">近 7 天总流量 (TB)</h2>
          <ResponsiveContainer width="100%" height={220}>
            <AreaChart data={trafficData}>
              <defs>
                <linearGradient id="tGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--color-primary)" stopOpacity={0.2} />
                  <stop offset="95%" stopColor="var(--color-primary)" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" vertical={false} />
              <XAxis dataKey="day" tick={{ fontSize: 12, fill: "var(--color-muted-foreground)" }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 12, fill: "var(--color-muted-foreground)" }} axisLine={false} tickLine={false} unit=" TB" width={50} />
              <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => [`${v} TB`, "流量"]} />
              <Area type="monotone" dataKey="tb" stroke="var(--color-primary)" strokeWidth={2} fill="url(#tGrad)" />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Recent users */}
      <div className="rounded-2xl border border-border/50 bg-card overflow-hidden card-shadow">
        <div className="px-5 sm:px-6 py-4 border-b border-border flex items-center justify-between">
          <h2 className="font-semibold text-lg text-foreground">最新注册用户</h2>
          <a href="/admin/users" className="text-sm text-primary hover:underline font-medium">查看全部</a>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-muted/50 border-b border-border">
                <th className="text-left px-5 sm:px-6 py-3 text-muted-foreground font-medium">邮箱</th>
                <th className="text-left px-5 sm:px-6 py-3 text-muted-foreground font-medium hidden sm:table-cell">套餐</th>
                <th className="text-left px-5 sm:px-6 py-3 text-muted-foreground font-medium hidden md:table-cell">注册时间</th>
                <th className="text-left px-5 sm:px-6 py-3 text-muted-foreground font-medium">状态</th>
              </tr>
            </thead>
            <tbody>
              {recentUsers.map((u) => (
                <tr key={u.email} className="border-b border-border/50 hover:bg-accent/50 transition-colors">
                  <td className="px-5 sm:px-6 py-4 text-foreground font-medium">{u.email}</td>
                  <td className="px-5 sm:px-6 py-4 hidden sm:table-cell">
                    <span className="text-xs rounded-lg bg-primary/10 text-primary px-2.5 py-1 font-medium">{u.plan}</span>
                  </td>
                  <td className="px-5 sm:px-6 py-4 text-muted-foreground hidden md:table-cell">{u.joined}</td>
                  <td className="px-5 sm:px-6 py-4">
                    <span className={`text-xs rounded-full px-3 py-1.5 font-medium ${
                      u.status === "active" ? "bg-success/10 text-success" : "bg-muted text-muted-foreground"
                    }`}>
                      {u.status === "active" ? "正常" : "已到期"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
