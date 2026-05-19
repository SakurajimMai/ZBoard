"use client"

import { useState } from "react"
import Link from "next/link"
import { Check, Sparkles } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/lib/i18n/context"

const plansData = [
  {
    key: "lite" as const,
    monthlyPrice: 15,
    yearlyPrice: 12,
    traffic: "100 GB / 月",
    devices: 3,
    speed: "100 Mbps",
    popular: false,
  },
  {
    key: "standard" as const,
    monthlyPrice: 30,
    yearlyPrice: 24,
    traffic: "300 GB / 月",
    devices: 5,
    speed: "500 Mbps",
    popular: true,
  },
  {
    key: "pro" as const,
    monthlyPrice: 60,
    yearlyPrice: 48,
    traffic: "1 TB / 月",
    devices: 10,
    speed: "∞",
    popular: false,
  },
]

export default function Pricing() {
  const [yearly, setYearly] = useState(false)
  const { t } = useI18n()

  return (
    <section id="pricing" className="py-20 sm:py-28 px-4 sm:px-6 lg:px-8 bg-muted/30">
      <div className="mx-auto max-w-5xl">
        <div className="text-center mb-12">
          <div className="inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/5 px-4 py-2 text-sm text-primary font-medium mb-4">
            {t.pricing.title}
          </div>
          <h2 className="text-balance text-3xl sm:text-4xl md:text-5xl font-bold text-foreground mb-4">
            {t.pricing.title}
          </h2>
          <p className="text-pretty text-base sm:text-lg text-muted-foreground max-w-xl mx-auto mb-8 leading-relaxed">
            {t.pricing.subtitle}
          </p>

          {/* Toggle */}
          <div className="inline-flex items-center gap-1 rounded-full border border-border bg-card p-1.5 card-shadow">
            <button
              onClick={() => setYearly(false)}
              className={`px-5 py-2 rounded-full text-sm font-medium transition-all ${
                !yearly ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {t.pricing.monthly}
            </button>
            <button
              onClick={() => setYearly(true)}
              className={`px-5 py-2 rounded-full text-sm font-medium transition-all flex items-center gap-2 ${
                yearly ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {t.pricing.yearly}
              <span className={`text-xs px-1.5 py-0.5 rounded-full ${yearly ? "bg-primary-foreground/20" : "bg-success/10 text-success"}`}>
                {t.pricing.save}
              </span>
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
          {plansData.map((plan) => {
            const planT = t.pricing.plans[plan.key]
            return (
              <div
                key={plan.key}
                className={`rounded-2xl border p-6 sm:p-8 flex flex-col transition-all relative ${
                  plan.popular
                    ? "border-primary/30 bg-card card-shadow-hover scale-[1.02]"
                    : "border-border/50 bg-card card-shadow hover:card-shadow-hover"
                }`}
              >
                {plan.popular && (
                  <div className="absolute -top-3 left-1/2 -translate-x-1/2 flex items-center gap-1.5 rounded-full bg-primary px-4 py-1.5 text-xs font-medium text-primary-foreground shadow-md">
                    <Sparkles className="w-3 h-3" />
                    {t.pricing.popular}
                  </div>
                )}
                <div className="mb-6">
                  <h3 className="font-bold text-lg text-foreground mb-1">{planT.name}</h3>
                  <p className="text-sm text-muted-foreground">{planT.desc}</p>
                  <div className="flex items-baseline gap-1 mt-4">
                    <span className="text-4xl sm:text-5xl font-bold text-foreground">
                      ¥{yearly ? plan.yearlyPrice : plan.monthlyPrice}
                    </span>
                    <span className="text-muted-foreground text-sm">{t.pricing.per_month}</span>
                  </div>
                  {yearly && (
                    <p className="text-xs text-success font-medium mt-2">¥{plan.yearlyPrice * 12} / 年</p>
                  )}
                </div>

                <div className="space-y-3 mb-6 text-sm">
                  {[
                    { label: "流量", value: plan.traffic },
                    { label: "速度", value: plan.speed },
                    { label: "设备", value: `${plan.devices}` },
                  ].map((row) => (
                    <div key={row.label} className="flex justify-between py-2 border-b border-border/50">
                      <span className="text-muted-foreground">{row.label}</span>
                      <span className="text-foreground font-medium">{row.value}</span>
                    </div>
                  ))}
                </div>

                <div className="mb-8 flex-1">
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-3">{t.pricing.features_inc}</p>
                  <ul className="space-y-2.5">
                    {[
                      plan.key === "lite" ? "基础节点访问" : "全部节点 + IPLC 专线",
                      "Vmess / Vless / Trojan",
                      `${plan.devices} 设备同时在线`,
                      plan.key === "lite" ? "邮件支持" : plan.key === "standard" ? "优先工单支持" : "7×24 专属客服",
                      ...(plan.key !== "lite" ? ["流量结转"] : []),
                    ].map((f) => (
                      <li key={f} className="flex items-start gap-3 text-sm">
                        <div className="w-5 h-5 rounded-full bg-success/10 flex items-center justify-center flex-shrink-0 mt-0.5">
                          <Check className="w-3 h-3 text-success" />
                        </div>
                        <span className="text-muted-foreground">{f}</span>
                      </li>
                    ))}
                  </ul>
                </div>

                <Link href="/dashboard">
                  <Button
                    className={`w-full h-11 font-medium ${
                      plan.popular
                        ? "btn-gradient text-primary-foreground shadow-md hover:shadow-lg"
                        : "border-border hover:border-primary/30 hover:bg-accent"
                    }`}
                    variant={plan.popular ? "default" : "outline"}
                  >
                    {t.pricing.subscribe}
                  </Button>
                </Link>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}
