"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { Check } from "lucide-react"
import { Button } from "@/components/ui/button"
import { getPlans } from "@/lib/api"

export default function Pricing() {
  const [plans, setPlans] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    getPlans()
      .then((res) => setPlans(res.items || []))
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <section id="pricing" className="py-20 px-4">
        <div className="max-w-6xl mx-auto text-center">
          <h2 className="text-3xl font-bold">选择套餐</h2>
          <p className="text-muted-foreground mt-2">加载中...</p>
        </div>
      </section>
    )
  }

  if (plans.length === 0) {
    return (
      <section id="pricing" className="py-20 px-4">
        <div className="max-w-6xl mx-auto text-center">
          <h2 className="text-3xl font-bold">选择套餐</h2>
          <p className="text-muted-foreground mt-2">暂无可用套餐</p>
        </div>
      </section>
    )
  }

  return (
    <section id="pricing" className="py-20 px-4">
      <div className="max-w-6xl mx-auto">
        <div className="text-center mb-12">
          <h2 className="text-3xl font-bold">选择套餐</h2>
          <p className="text-muted-foreground mt-2">灵活的方案满足不同需求</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 max-w-4xl mx-auto">
          {plans.map((plan, i) => (
            <div
              key={plan.id}
              className={`rounded-2xl border p-6 flex flex-col ${
                i === 1 ? "border-primary shadow-lg scale-[1.02]" : ""
              }`}
            >
              <h3 className="text-xl font-semibold">{plan.name}</h3>
              <div className="mt-4 flex items-baseline gap-1">
                <span className="text-4xl font-bold">¥{plan.price}</span>
                <span className="text-muted-foreground">/ {plan.duration_days}天</span>
              </div>

              <ul className="mt-6 space-y-3 flex-1">
                {featureList(plan).map((feature) => (
                  <li key={feature} className="flex items-center gap-2 text-sm">
                    <Check className="w-4 h-4 text-green-500 flex-shrink-0" />
                    {feature}
                  </li>
                ))}
              </ul>

              <Link href="/login" className="mt-6">
                <Button className="w-full" variant={i === 1 ? "default" : "outline"}>
                  立即购买
                </Button>
              </Link>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

function featureList(plan: any): string[] {
  const explicit = Array.isArray(plan.features) ? plan.features.filter(Boolean) : []
  if (explicit.length > 0) return explicit
  const features = [
    `${(plan.traffic_limit / 1073741824).toFixed(0)} GB 流量`,
    `${plan.device_limit} 台设备同时在线`,
  ]
  features.push("全部节点可用", "支持 Clash / sing-box / V2rayN")
  return features
}
