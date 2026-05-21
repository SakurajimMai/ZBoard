"use client"

import { useEffect, useState } from "react"
import { CreditCard, ShoppingCart, Check } from "lucide-react"
import { Button } from "@/components/ui/button"
import { getPlans, createOrder, payOrder, getPaymentMethods } from "@/lib/api"

export default function Billing() {
  const [plans, setPlans] = useState<any[]>([])
  const [methods, setMethods] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [buying, setBuying] = useState<number | null>(null)

  useEffect(() => {
    Promise.all([getPlans(), getPaymentMethods()])
      .then(([plansRes, methodsRes]) => {
        setPlans(plansRes.items || [])
        setMethods(methodsRes.methods || [])
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  const handleBuy = async (planId: number) => {
    setBuying(planId)
    try {
      const orderRes = await createOrder(planId)
      const orderNo = orderRes.order.order_no

      // Use first available payment method, or mock
      const provider = methods.length > 0 ? methods[0].name : undefined
      const payRes = await payOrder(orderNo, provider)

      if (payRes.pay_url) {
        window.location.href = payRes.pay_url
      } else {
        alert("订单已创建: " + orderNo)
      }
    } catch (err: any) {
      alert(err.message || "购买失败")
    } finally {
      setBuying(null)
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">套餐购买</h1>
        <p className="text-sm text-muted-foreground mt-1">选择适合您的套餐方案。</p>
      </div>

      {/* Payment methods info */}
      {methods.length > 0 && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <CreditCard className="w-4 h-4" />
          <span>支持：{methods.map(m => m.display_name || m.name).join("、")}</span>
        </div>
      )}

      {/* Plans */}
      {plans.length === 0 ? (
        <div className="text-muted-foreground">暂无可用套餐</div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {plans.map((plan) => (
            <div key={plan.id} className="rounded-xl border bg-card p-5 flex flex-col">
              <h3 className="font-semibold text-lg">{plan.name}</h3>
              <div className="mt-2 flex items-baseline gap-1">
                <span className="text-3xl font-bold">¥{plan.price}</span>
                <span className="text-sm text-muted-foreground">/ {plan.duration_days}天</span>
              </div>
              <ul className="mt-4 space-y-2 text-sm text-muted-foreground flex-1">
                {featureList(plan).map((feature) => (
                  <li key={feature} className="flex items-center gap-2">
                    <Check className="w-4 h-4 text-green-500" />
                    {feature}
                  </li>
                ))}
              </ul>
              <Button
                className="mt-4 w-full"
                onClick={() => handleBuy(plan.id)}
                disabled={buying === plan.id}
              >
                <ShoppingCart className="w-4 h-4 mr-1" />
                {buying === plan.id ? "处理中..." : "立即购买"}
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function featureList(plan: any): string[] {
  const explicit = Array.isArray(plan.features) ? plan.features.filter(Boolean) : []
  if (explicit.length > 0) return explicit
  const features = [
    `流量 ${(plan.traffic_limit / 1073741824).toFixed(0)} GB`,
    `${plan.device_limit} 台设备同时在线`,
  ]
  features.push("全部节点可用")
  return features
}
