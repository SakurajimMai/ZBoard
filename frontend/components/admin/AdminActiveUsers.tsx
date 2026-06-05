"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { Activity, CalendarDays, Clock3, Download, Network, Upload } from "lucide-react"
import { AdminPager } from "@/components/admin/AdminPager"
import { Button } from "@/components/ui/button"
import { adminGetActiveUsers } from "@/lib/api"

type ActiveUser = {
  user_id: number
  email: string
  status: string
  plan_id: number | null
  plan_name?: string | null
  upload_total: number
  download_total: number
  traffic_total: number
  node_count: number
  first_active_at: string
  last_active_at: string
}

type ActiveSummary = {
  active_users: number
  upload_total: number
  download_total: number
  traffic_total: number
}

function lastSevenDays() {
  const days: { value: string; label: string; weekday: string; isToday: boolean }[] = []
  const now = new Date()
  for (let i = 0; i < 7; i++) {
    const d = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() - i))
    const value = d.toISOString().slice(0, 10)
    days.push({
      value,
      label: `${d.getUTCMonth() + 1}/${d.getUTCDate()}`,
      weekday: ["周日", "周一", "周二", "周三", "周四", "周五", "周六"][d.getUTCDay()],
      isToday: i === 0,
    })
  }
  return days
}

function formatTrafficBytes(value: unknown) {
  const bytes = Number(value || 0)
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB", "PB"]
  let n = bytes
  let i = 0
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i += 1
  }
  const fixed = i === 0 ? 0 : n >= 100 ? 0 : n >= 10 ? 1 : 2
  return `${n.toFixed(fixed)} ${units[i]}`
}

function formatTime(value: string) {
  if (!value) return "-"
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return "-"
  return d.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit", second: "2-digit" })
}

export default function AdminActiveUsers() {
  const days = useMemo(() => lastSevenDays(), [])
  const [selectedDate, setSelectedDate] = useState(days[0]?.value || "")
  const [items, setItems] = useState<ActiveUser[]>([])
  const [summary, setSummary] = useState<ActiveSummary>({
    active_users: 0,
    upload_total: 0,
    download_total: 0,
    traffic_total: 0,
  })
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)

  const load = useCallback(() => {
    setLoading(true)
    adminGetActiveUsers({ date: selectedDate, page, pageSize })
      .then((res) => {
        setItems(res.items || [])
        setSummary(res.summary || {
          active_users: 0,
          upload_total: 0,
          download_total: 0,
          traffic_total: 0,
        })
        setTotal(res.total || 0)
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [page, pageSize, selectedDate])

  useEffect(() => {
    load()
  }, [load])

  const selectedDay = days.find((d) => d.value === selectedDate)

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-2xl font-bold">活跃用户</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            按真实流量统计，当天有上传或下载增量的用户会计入活跃。
          </p>
        </div>
        <Button size="sm" variant="outline" onClick={load} disabled={loading}>
          <Activity className="mr-1 h-4 w-4" />
          刷新
        </Button>
      </div>

      <div className="grid gap-3 md:grid-cols-4">
        <SummaryCard icon={Activity} label="活跃用户" value={`${summary.active_users}`} hint={selectedDay?.isToday ? "今天" : selectedDate} />
        <SummaryCard icon={Network} label="总流量" value={formatTrafficBytes(summary.traffic_total)} hint="上传 + 下载" />
        <SummaryCard icon={Download} label="下载" value={formatTrafficBytes(summary.download_total)} hint="所选日期" />
        <SummaryCard icon={Upload} label="上传" value={formatTrafficBytes(summary.upload_total)} hint="所选日期" />
      </div>

      <div className="rounded-lg border bg-card p-3">
        <div className="mb-3 flex items-center gap-2 text-sm font-medium">
          <CalendarDays className="h-4 w-4 text-primary" />
          最近 7 天
        </div>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-7">
          {days.map((day) => (
            <Button
              key={day.value}
              type="button"
              variant={selectedDate === day.value ? "default" : "outline"}
              className="h-auto flex-col items-start gap-1 px-3 py-2"
              onClick={() => {
                setSelectedDate(day.value)
                setPage(1)
              }}
            >
              <span className="text-sm font-semibold">{day.isToday ? "今天" : day.label}</span>
              <span className="text-xs opacity-70">{day.weekday}</span>
            </Button>
          ))}
        </div>
      </div>

      <div className="rounded-lg border bg-card overflow-hidden">
        <div className="border-b px-4 py-3">
          <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
            <h2 className="font-semibold">真实活跃用户列表</h2>
            <span className="text-sm text-muted-foreground">{selectedDate} · 共 {total} 人</span>
          </div>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">用户</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">状态</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground hidden md:table-cell">套餐</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">当天流量</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground hidden lg:table-cell">上传 / 下载</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground hidden lg:table-cell">节点数</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">活跃时间</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center text-muted-foreground">加载中...</td>
                </tr>
              ) : items.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center text-muted-foreground">该日期暂无活跃用户</td>
                </tr>
              ) : items.map((item) => (
                <tr key={item.user_id} className="border-b hover:bg-accent/50">
                  <td className="px-4 py-3">
                    <div className="font-medium">{item.email}</div>
                    <div className="text-xs text-muted-foreground">ID {item.user_id}</div>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`rounded-full px-2 py-1 text-xs ${
                      item.status === "active" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"
                    }`}>
                      {item.status === "active" ? "正常" : "已禁用"}
                    </span>
                  </td>
                  <td className="px-4 py-3 hidden md:table-cell text-muted-foreground">
                    {item.plan_name || (item.plan_id ? `套餐 #${item.plan_id}` : "-")}
                  </td>
                  <td className="px-4 py-3 font-semibold">{formatTrafficBytes(item.traffic_total)}</td>
                  <td className="px-4 py-3 hidden lg:table-cell text-muted-foreground">
                    上 {formatTrafficBytes(item.upload_total)} / 下 {formatTrafficBytes(item.download_total)}
                  </td>
                  <td className="px-4 py-3 hidden lg:table-cell text-muted-foreground">{item.node_count}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1 text-muted-foreground">
                      <Clock3 className="h-3.5 w-3.5" />
                      <span>{formatTime(item.first_active_at)} - {formatTime(item.last_active_at)}</span>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <AdminPager
        page={page}
        pageSize={pageSize}
        total={total}
        onPageChange={setPage}
        onPageSizeChange={(size) => {
          setPageSize(size)
          setPage(1)
        }}
      />
    </div>
  )
}

function SummaryCard({ icon: Icon, label, value, hint }: { icon: any; label: string; value: string; hint: string }) {
  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="mb-3 flex items-center justify-between">
        <span className="text-sm text-muted-foreground">{label}</span>
        <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary/10">
          <Icon className="h-4 w-4 text-primary" />
        </div>
      </div>
      <div className="text-2xl font-bold">{value}</div>
      <div className="mt-1 text-xs text-muted-foreground">{hint}</div>
    </div>
  )
}
