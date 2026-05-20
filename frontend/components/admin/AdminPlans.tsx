"use client"

import { useEffect, useState } from "react"
import { adminGetPlans } from "@/lib/api"

export default function AdminPlans() {
  const [plans, setPlans] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    adminGetPlans()
      .then((res) => setPlans(res.items || []))
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">套餐管理</h1>
        <p className="text-sm text-muted-foreground mt-1">共 {plans.length} 个套餐</p>
      </div>

      {plans.length === 0 ? (
        <div className="text-center text-muted-foreground py-12">暂无套餐，请通过 API 创建</div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {plans.map((p: any) => (
            <div key={p.id} className="rounded-xl border bg-card p-5">
              <div className="flex items-center justify-between mb-2">
                <h3 className="font-semibold">{p.name}</h3>
                <span className={`text-xs rounded-full px-2 py-0.5 ${
                  p.status === "active" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-600"
                }`}>
                  {p.status === "active" ? "上架" : "下架"}
                </span>
              </div>
              <div className="text-2xl font-bold">¥{p.price}</div>
              <div className="text-sm text-muted-foreground mt-1">{p.duration_days} 天</div>
              <div className="mt-3 text-xs text-muted-foreground space-y-1">
                <p>流量: {(p.traffic_limit / 1073741824).toFixed(0)} GB</p>
                <p>设备: {p.device_limit} 台</p>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
