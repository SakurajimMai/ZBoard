"use client"

import { useEffect, useState } from "react"
import { CreditCard, ShoppingCart, Check, ShieldCheck, HelpCircle } from "lucide-react"
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
        alert("订单同步接入凭证已生成，订单编号: " + orderNo)
      }
    } catch (err: any) {
      alert(err.message || "服务升级失败，请重试")
    } finally {
      setBuying(null)
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">正在载入高速服务方案...</div>

  return (
    <div className="space-y-8 select-none">
      {/* 头部标题 */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 border-b border-border/60 pb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">升级加速方案</h1>
          <p className="text-sm text-muted-foreground mt-1.5">选择契合您业务需要的云同步加速与协同方案。</p>
        </div>
        {methods.length > 0 && (
          <div className="flex items-center gap-2 text-xs bg-secondary/50 border rounded-full px-4.5 py-2 font-medium text-muted-foreground">
            <CreditCard className="w-3.5 h-3.5 text-blue-500" />
            <span>支付支持：{methods.map(m => m.display_name || m.name).join("、")}</span>
          </div>
        )}
      </div>

      {/* Plans 套餐矩阵 */}
      {plans.length === 0 ? (
        <div className="text-center text-muted-foreground py-16 border border-dashed rounded-2xl bg-secondary/20">
          <HelpCircle className="w-10 h-10 mx-auto text-muted-foreground/60 mb-2.5" />
          <p className="text-sm">当前暂无发布的可用服务网点方案</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {plans.map((plan) => {
            // 是否是推荐主推套餐 (比如包含“标准”或“日常”的套餐，或者是中等价格套餐)
            const isPopular = plan.name.includes("标准") || plan.name.includes("Standard") || plan.name.includes("日常")
            
            return (
              <div
                key={plan.id}
                className={`relative rounded-2xl transition-all duration-500 flex flex-col group ${
                  isPopular
                    ? "p-[1px] bg-gradient-to-tr from-blue-500 via-indigo-500 to-sky-400 shadow-[0_20px_50px_rgba(59,130,246,0.12)] scale-[1.02] z-10"
                    : "border border-border/80 bg-card shadow-[0_8px_30px_rgb(0,0,0,0.02)] hover:shadow-[0_20px_40px_rgba(0,0,0,0.06)] hover:-translate-y-1"
                }`}
              >
                {/* 推荐徽章 */}
                {isPopular && (
                  <span className="absolute -top-3 right-6 text-[10px] font-bold text-white bg-gradient-to-r from-blue-600 to-indigo-600 rounded-full px-3.5 py-1 uppercase tracking-wider shadow-md shadow-blue-500/10">
                    热门主推
                  </span>
                )}

                <div className={`rounded-2xl bg-card p-6 sm:p-7 flex flex-col flex-1 ${isPopular ? "h-full" : ""}`}>
                  {/* 套餐名称 */}
                  <h3 className="font-bold text-lg text-foreground group-hover:text-blue-500 transition-colors">
                    {plan.name}
                  </h3>
                  
                  {/* 价格区块 */}
                  <div className="mt-3 flex items-baseline gap-1 border-b border-secondary/60 pb-5">
                    <span className="text-3xl font-extrabold tracking-tight bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent">
                      ¥{plan.price}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      / {plan.duration_days}天使用周期
                    </span>
                  </div>

                  {/* 特性展示 */}
                  <ul className="mt-5 space-y-3.5 text-sm text-slate-600 dark:text-slate-400 flex-1">
                    {featureList(plan).map((feature) => (
                      <li key={feature} className="flex items-start gap-2.5 leading-tight">
                        <ShieldCheck className="w-4.5 h-4.5 text-blue-500 shrink-0 mt-0.5" />
                        <span>{feature}</span>
                      </li>
                    ))}
                  </ul>

                  {/* 提交升级购买按钮 */}
                  <Button
                    onClick={() => handleBuy(plan.id)}
                    disabled={buying === plan.id}
                    className={`mt-6 w-full rounded-xl py-3.5 font-semibold flex items-center justify-center gap-2.5 transition-all duration-300 ${
                      isPopular
                        ? "bg-gradient-to-r from-blue-600 to-indigo-600 hover:opacity-95 text-white shadow-md shadow-blue-500/15 active:scale-[0.98]"
                        : "bg-secondary text-secondary-foreground hover:bg-secondary/80 border border-border/50 active:scale-[0.98]"
                    }`}
                  >
                    <ShoppingCart className="w-4 h-4" />
                    <span>{buying === plan.id ? "正在创建同步通道..." : "立即部署方案"}</span>
                  </Button>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

function featureList(plan: any): string[] {
  const explicit = Array.isArray(plan.features) ? plan.features.filter(Boolean) : []
  
  // 对外部录入的功能清单进行脱敏替换
  const sanitize = (feat: string) => {
    return feat
      .replace(/全部节点/g, "全部高速服务网点")
      .replace(/节点/g, "加速网点")
      .replace(/客户端/g, "网络工具")
      .replace(/订阅/g, "同步配置")
      .replace(/翻墙/g, "网络加速")
      .replace(/代理/g, "网络协同")
  }

  if (explicit.length > 0) return explicit.map(sanitize)

  // 默认生成的清单
  return [
    `数据流同步配额 ${(plan.traffic_limit / 1073741824).toFixed(0)} GB / 周期`,
    `支持 ${plan.device_limit} 台多终端设备同步在线`,
    "支持接入全部高速服务网点",
    "采用端到端高强度防劫持数据加密",
  ]
}
