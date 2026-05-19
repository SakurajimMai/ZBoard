"use client"

import { CreditCard, ShoppingCart, ArrowUpRight, Check } from "lucide-react"
import { Button } from "@/components/ui/button"

const orders = [
  { id: "ORD-20240518-001", plan: "标准版 月付", amount: 30, date: "2024-05-18", status: "paid", method: "支付宝" },
  { id: "ORD-20240418-002", plan: "标准版 月付", amount: 30, date: "2024-04-18", status: "paid", method: "微信支付" },
  { id: "ORD-20240318-003", plan: "入门版 月付", amount: 15, date: "2024-03-18", status: "paid", method: "支付宝" },
  { id: "ORD-20240218-004", plan: "流量包 50GB", amount: 10, date: "2024-02-18", status: "paid", method: "支付宝" },
]

const packs = [
  { name: "50 GB 流量包", price: 10, popular: false },
  { name: "100 GB 流量包", price: 18, popular: true },
  { name: "200 GB 流量包", price: 30, popular: false },
]

export default function Billing() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">账单 & 充值</h1>
        <p className="text-sm text-muted-foreground mt-1">管理您的订阅、购买流量包和查看历史账单。</p>
      </div>

      {/* Current plan */}
      <div className="rounded-xl border border-primary/40 bg-primary/5 p-5 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <span className="text-sm font-medium text-foreground">当前套餐</span>
            <span className="text-xs rounded-full bg-primary/20 text-primary px-2 py-0.5">激活中</span>
          </div>
          <p className="text-2xl font-bold text-foreground">标准版</p>
          <p className="text-sm text-muted-foreground mt-0.5">¥30/月 · 到期：2024-06-18 · 剩余流量：212.6 GB</p>
        </div>
        <div className="flex gap-3 flex-wrap">
          <Button variant="outline" className="gap-2 hover:border-primary/50">
            <ArrowUpRight className="w-4 h-4" />
            升级套餐
          </Button>
          <Button className="gap-2 bg-primary text-primary-foreground hover:bg-primary/90">
            <CreditCard className="w-4 h-4" />
            立即续费
          </Button>
        </div>
      </div>

      {/* Traffic packs */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h2 className="font-semibold text-foreground mb-4">购买流量包</h2>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          {packs.map((pack) => (
            <div
              key={pack.name}
              className={`rounded-xl border p-4 relative ${
                pack.popular ? "border-primary bg-primary/5" : "border-border hover:border-primary/30"
              } transition-colors`}
            >
              {pack.popular && (
                <span className="absolute -top-2.5 left-4 rounded-full bg-primary px-3 py-0.5 text-xs text-primary-foreground">
                  热门
                </span>
              )}
              <p className="font-medium text-foreground mb-1">{pack.name}</p>
              <p className="text-2xl font-bold text-foreground mb-3">¥{pack.price}</p>
              <Button
                size="sm"
                className={`w-full gap-1.5 ${
                  pack.popular
                    ? "bg-primary text-primary-foreground"
                    : "border-border hover:border-primary/50"
                }`}
                variant={pack.popular ? "default" : "outline"}
              >
                <ShoppingCart className="w-3.5 h-3.5" />
                立即购买
              </Button>
            </div>
          ))}
        </div>
      </div>

      {/* Order history */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <h2 className="font-semibold text-foreground">订单历史</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-secondary/50 border-b border-border">
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">订单号</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">项目</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">金额</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">支付方式</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">日期</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">状态</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((o) => (
                <tr key={o.id} className="border-b border-border/50 hover:bg-accent/50 transition-colors">
                  <td className="px-5 py-3 font-mono text-xs text-muted-foreground">{o.id}</td>
                  <td className="px-5 py-3 text-foreground">{o.plan}</td>
                  <td className="px-5 py-3 font-semibold text-foreground">¥{o.amount}</td>
                  <td className="px-5 py-3 text-muted-foreground">{o.method}</td>
                  <td className="px-5 py-3 text-muted-foreground">{o.date}</td>
                  <td className="px-5 py-3">
                    <span className="inline-flex items-center gap-1.5 text-xs text-green-400">
                      <Check className="w-3 h-3" />
                      已支付
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
