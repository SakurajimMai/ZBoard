"use client"

import { Shield, Zap, Globe, Lock, RefreshCw, Headphones, MonitorSmartphone, BarChart3 } from "lucide-react"
import { useI18n } from "@/lib/i18n/context"

const featureIcons = [Zap, Globe, Shield, Lock, RefreshCw, MonitorSmartphone, BarChart3, Headphones]
const featureKeys = ["speed", "nodes", "protocol", "safe", "stable", "device", "price", "support"] as const

export default function Features() {
  const { t } = useI18n()

  const features = featureKeys.map((key, i) => ({
    icon: featureIcons[i],
    title: t.features.items[key].title,
    desc:  t.features.items[key].desc,
  }))

  return (
    <section id="features" className="py-20 sm:py-28 px-4 sm:px-6 lg:px-8 bg-background">
      <div className="mx-auto max-w-7xl">
        {/* Header */}
        <div className="text-center mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/5 px-4 py-2 text-sm text-primary font-medium mb-4">
            {t.features.title}
          </div>
          <h2 className="text-balance text-3xl sm:text-4xl md:text-5xl font-bold text-foreground mb-4">
            {t.features.title}
          </h2>
          <p className="text-pretty text-base sm:text-lg text-muted-foreground max-w-xl mx-auto leading-relaxed">
            {t.features.subtitle}
          </p>
        </div>

        {/* Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-5">
          {features.map((f) => {
            const Icon = f.icon
            return (
              <div
                key={f.title}
                className="group rounded-2xl border border-border/50 bg-card p-6 card-shadow hover:card-shadow-hover hover:border-primary/20 transition-all duration-300"
              >
                <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mb-5 group-hover:bg-primary/15 transition-colors">
                  <Icon className="w-6 h-6 text-primary" />
                </div>
                <h3 className="font-semibold text-foreground text-lg mb-2">{f.title}</h3>
                <p className="text-sm text-muted-foreground leading-relaxed">{f.desc}</p>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}
