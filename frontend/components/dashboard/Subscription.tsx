"use client"

import { useEffect, useState } from "react"
import { Wifi, Download, Upload, Globe } from "lucide-react"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { getMe, getTrafficLogs, getUserNodes } from "@/lib/api"
import {
  Area, AreaChart, CartesianGrid, XAxis, YAxis, Tooltip, ResponsiveContainer, Legend
} from "recharts"

function formatTime(dateStr: string) {
  const d = new Date(dateStr)
  return `${d.getHours().toString().padStart(2, "0")}:${d.getMinutes().toString().padStart(2, "0")}`
}

function bytesToMB(bytes: number) {
  return (bytes / 1048576).toFixed(3)
}

export default function Subscription() {
  const [user, setUser] = useState<any>(null)
  const [trafficLogs, setTrafficLogs] = useState<any[]>([])
  const [nodes, setNodes] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [timeRange, setTimeRange] = useState("30min")

  useEffect(() => {
    Promise.all([
      getMe(),
      getTrafficLogs(200).catch(() => ({ items: [] })),
      getUserNodes().catch(() => ({ items: [] })),
    ])
      .then(([meRes, logsRes, nodesRes]) => {
        setUser(meRes.user)
        setTrafficLogs(logsRes.items || [])
        setNodes(nodesRes.items || [])
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  // Filter logs by time range
  const now = Date.now()
  const rangeMs = timeRange === "30min" ? 30 * 60000 : timeRange === "1h" ? 60 * 60000 : 120 * 60000
  const filteredLogs = trafficLogs
    .filter((log) => {
      const t = new Date(log.reported_at).getTime()
      return now - t <= rangeMs
    })
    .reverse()

  const chartData = filteredLogs.map((log) => ({
    time: formatTime(log.reported_at),
    download: Number(bytesToMB(log.download_delta)),
    upload: Number(bytesToMB(log.upload_delta)),
  }))

  const regionMap: Record<string, string> = {
    HK: "🇭🇰 HK",
    JP: "🇯🇵 JP",
    TW: "🇹🇼 TW",
    SG: "🇸🇬 SG",
    US: "🇺🇸 US",
    KR: "🇰🇷 KR",
    DE: "🇩🇪 DE",
    UK: "🇬🇧 UK",
  }

  return (
    <div className="space-y-6">
      {/* === 流量图表 === */}
      <div className="rounded-xl border bg-card p-5">
        <Tabs defaultValue="recent">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4">
            <TabsList>
              <TabsTrigger value="recent">近期流量</TabsTrigger>
              <TabsTrigger value="daily">每日流量</TabsTrigger>
              <TabsTrigger value="logs">同步日志</TabsTrigger>
            </TabsList>
            <div className="flex gap-1">
              {(["30min", "1h", "2h"] as const).map((r) => (
                <Button
                  key={r}
                  size="sm"
                  variant={timeRange === r ? "default" : "outline"}
                  onClick={() => setTimeRange(r)}
                  className="text-xs px-3"
                >
                  {r === "30min" ? "30 分钟" : r === "1h" ? "1 小时" : "2 小时"}
                </Button>
              ))}
            </div>
          </div>

          <TabsContent value="recent" className="space-y-6">
            <h3 className="text-lg font-bold">近期流量</h3>

            {/* 下载流量图 */}
            <div>
              <p className="text-sm font-medium mb-2 flex items-center gap-2">
                <span className="inline-block w-2 h-2 rounded-full bg-red-500"></span>
                近期下载流量 <span className="text-xs text-muted-foreground font-normal">下载</span>
              </p>
              <div className="h-[200px] w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={chartData}>
                    <defs>
                      <linearGradient id="dlGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                    <XAxis dataKey="time" tick={{ fontSize: 11 }} />
                    <YAxis tick={{ fontSize: 11 }} unit=" MB" />
                    <Tooltip formatter={(v: number) => [`${v} MB`, "Download"]} />
                    <Legend />
                    <Area
                      type="monotone" dataKey="download" name="Download (MB)"
                      stroke="#3b82f6" fill="url(#dlGradient)" strokeWidth={2}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>

            {/* 上传流量图 */}
            <div>
              <p className="text-sm font-medium mb-2 flex items-center gap-2">
                <span className="inline-block w-2 h-2 rounded-full bg-blue-400"></span>
                近期上传流量 <span className="text-xs text-muted-foreground font-normal">上传</span>
              </p>
              <div className="h-[200px] w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={chartData}>
                    <defs>
                      <linearGradient id="ulGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#38bdf8" stopOpacity={0.3} />
                        <stop offset="95%" stopColor="#38bdf8" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                    <XAxis dataKey="time" tick={{ fontSize: 11 }} />
                    <YAxis tick={{ fontSize: 11 }} unit=" MB" />
                    <Tooltip formatter={(v: number) => [`${v} MB`, "Upload"]} />
                    <Legend />
                    <Area
                      type="monotone" dataKey="upload" name="Upload (MB)"
                      stroke="#38bdf8" fill="url(#ulGradient)" strokeWidth={2}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="daily">
            <div className="text-center text-muted-foreground py-12">每日流量统计 — 即将推出</div>
          </TabsContent>

          <TabsContent value="logs">
            <div className="text-center text-muted-foreground py-12">同步日志 — 即将推出</div>
          </TabsContent>
        </Tabs>
      </div>

      {/* === 服务网点列表 === */}
      <div className="rounded-xl border bg-card p-5">
        <h3 className="text-lg font-bold mb-1">全球服务网点</h3>
        <p className="text-xs text-muted-foreground mb-4">查看您当前可用的加速网点</p>

        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left">
                <th className="py-3 px-2 font-medium text-muted-foreground">加速网点</th>
                <th className="py-3 px-2 font-medium text-muted-foreground">地区</th>
                <th className="py-3 px-2 font-medium text-muted-foreground text-right">网点状态</th>
              </tr>
            </thead>
            <tbody>
              {nodes.length === 0 ? (
                <tr>
                  <td colSpan={3} className="py-8 text-center text-muted-foreground">暂无可用加速网点</td>
                </tr>
              ) : (
                nodes.map((node) => {
                  const region = node.region || "—"
                  const regionLabel = regionMap[region.toUpperCase()] || `${region.toLowerCase()} ${region}`
                  return (
                    <tr key={node.id} className="border-b last:border-0 hover:bg-secondary/30 transition">
                      <td className="py-3 px-2 font-medium">{node.name}</td>
                      <td className="py-3 px-2 text-muted-foreground">{regionLabel}</td>
                      <td className="py-3 px-2 text-right">
                        <span className={`text-xs font-medium px-2 py-0.5 rounded ${
                          node.status === "online"
                            ? "text-green-700 bg-green-50"
                            : "text-red-600 bg-red-50"
                        }`}>
                          {node.status === "online" ? "在线" : "离线"}
                        </span>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
