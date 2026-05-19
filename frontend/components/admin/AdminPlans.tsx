"use client"

import { Plus, Edit, Trash2, Eye, EyeOff } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useState } from "react"

const plans = [
  {
    id: 1,
    name: "入门版",
    price: 15,
    traffic: 100,
    devices: 3,
    speed: "100 Mbps",
    subscribers: 3241,
    visible: true,
  },
  {
    id: 2,
    name: "标准版",
    price: 30,
    traffic: 300,
    devices: 5,
    speed: "500 Mbps",
    subscribers: 6892,
    visible: true,
  },
  {
    id: 3,
    name: "旗舰版",
    price: 60,
    traffic: 1024,
    devices: 10,
    speed: "不限速",
    subscribers: 2348,
    visible: true,
  },
  {
    id: 4,
    name: "体验版（已下架）",
    price: 5,
    traffic: 20,
    devices: 1,
    speed: "50 Mbps",
    subscribers: 0,
    visible: false,
  },
]

export default function AdminPlans() {
  const [planList, setPlanList] = useState(plans)

  const toggleVisible = (id: number) => {
    setPlanList((prev) => prev.map((p) => p.id === id ? { ...p, visible: !p.visible } : p))
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">套餐管理</h1>
          <p className="text-sm text-muted-foreground mt-1">管理对外展示的订阅套餐。</p>
        </div>
        <Button className="gap-2 bg-primary text-primary-foreground hover:bg-primary/90">
          <Plus className="w-4 h-4" />
          新建套餐
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {planList.map((plan) => (
          <div
            key={plan.id}
            className={`rounded-xl border p-5 ${
              plan.visible ? "border-border bg-card" : "border-border/40 bg-card/50 opacity-60"
            }`}
          >
            <div className="flex items-start justify-between mb-4">
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="font-semibold text-foreground">{plan.name}</h3>
                  <span className={`text-xs rounded-full px-2 py-0.5 ${
                    plan.visible ? "bg-green-500/15 text-green-400" : "bg-secondary text-muted-foreground"
                  }`}>
                    {plan.visible ? "已上架" : "已下架"}
                  </span>
                </div>
                <p className="text-2xl font-bold text-foreground mt-1">¥{plan.price}<span className="text-sm font-normal text-muted-foreground">/月</span></p>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => toggleVisible(plan.id)}
                  className="p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-primary transition-colors"
                  aria-label={plan.visible ? "下架套餐" : "上架套餐"}
                >
                  {plan.visible ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
                </button>
                <button className="p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors">
                  <Edit className="w-4 h-4" />
                </button>
                <button className="p-1.5 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors">
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 text-sm mb-4">
              <div className="rounded-lg bg-secondary p-3">
                <p className="text-xs text-muted-foreground mb-0.5">月流量</p>
                <p className="font-medium text-foreground">{plan.traffic} GB</p>
              </div>
              <div className="rounded-lg bg-secondary p-3">
                <p className="text-xs text-muted-foreground mb-0.5">设备数</p>
                <p className="font-medium text-foreground">{plan.devices} 台</p>
              </div>
              <div className="rounded-lg bg-secondary p-3">
                <p className="text-xs text-muted-foreground mb-0.5">速度限制</p>
                <p className="font-medium text-foreground">{plan.speed}</p>
              </div>
              <div className="rounded-lg bg-secondary p-3">
                <p className="text-xs text-muted-foreground mb-0.5">订阅人数</p>
                <p className="font-medium text-foreground">{plan.subscribers.toLocaleString()}</p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
