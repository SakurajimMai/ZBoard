"use client"

import { useEffect, useState } from "react"
import { adminGetOrders } from "@/lib/api"

export default function AdminOrdersPage() {
  const [orders, setOrders] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    adminGetOrders()
      .then((res) => setOrders(res.items || []))
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  const paidOrders = orders.filter((o) => o.status === "paid")
  const total = paidOrders.reduce((s, o) => s + parseFloat(o.amount || "0"), 0)

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">订单管理</h1>
        <p className="text-sm text-muted-foreground mt-1">
          共 {orders.length} 笔订单，已付 {paidOrders.length} 笔，总收入 ¥{total.toFixed(2)}
        </p>
      </div>

      {orders.length === 0 ? (
        <div className="text-center text-muted-foreground py-12">暂无订单</div>
      ) : (
        <div className="rounded-xl border bg-card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted/50 border-b">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">订单号</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden md:table-cell">用户ID</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">金额</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">创建时间</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">状态</th>
                </tr>
              </thead>
              <tbody>
                {orders.map((o: any) => (
                  <tr key={o.id} className="border-b hover:bg-accent/50">
                    <td className="px-4 py-3 font-mono text-xs">{o.order_no}</td>
                    <td className="px-4 py-3 hidden md:table-cell">{o.user_id}</td>
                    <td className="px-4 py-3 font-medium">¥{o.amount}</td>
                    <td className="px-4 py-3 hidden lg:table-cell text-muted-foreground">
                      {new Date(o.created_at).toLocaleString("zh-CN")}
                    </td>
                    <td className="px-4 py-3">
                      <span className={`text-xs rounded-full px-2 py-1 ${
                        o.status === "paid" ? "bg-green-100 text-green-700" :
                        o.status === "pending" ? "bg-yellow-100 text-yellow-700" :
                        "bg-gray-100 text-gray-600"
                      }`}>
                        {o.status === "paid" ? "已付" : o.status === "pending" ? "待付" : o.status}
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
