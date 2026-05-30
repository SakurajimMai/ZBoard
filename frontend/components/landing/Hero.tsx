"use client"

import Link from "next/link"
import { Button } from "@/components/ui/button"
import { ArrowRight, Shield, Zap, Globe, Play } from "lucide-react"
import { useI18n } from "@/lib/i18n/context"

export default function Hero() {
  const { t } = useI18n()

  const stats = [
    { label: t.hero.stat_users,  value: "10万+", icon: Globe },
    { label: t.hero.stat_nodes,  value: "200+",  icon: Shield },
    { label: t.hero.stat_uptime, value: "99.9%", icon: Play },
    { label: t.hero.stat_speed,  value: "10Gbps", icon: Zap },
  ]

  return (
    <section className="relative min-h-dvh flex items-center justify-center overflow-hidden pt-16 gradient-hero">
      {/* Decorative elements */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 -right-40 w-80 h-80 rounded-full bg-primary/5 blur-3xl" />
        <div className="absolute -bottom-40 -left-40 w-80 h-80 rounded-full bg-chart-2/5 blur-3xl" />
        <div
          className="absolute inset-0 opacity-[0.015]"
          style={{
            backgroundImage: `radial-gradient(circle at 1px 1px, currentColor 1px, transparent 0)`,
            backgroundSize: "40px 40px",
          }}
        />
      </div>

      <div className="relative z-10 mx-auto max-w-6xl px-4 sm:px-6 lg:px-8 py-12 sm:py-20">
        <div className="text-center">
          {/* Badge */}
          <div className="inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/5 px-4 py-2 text-sm text-primary font-medium mb-8 shadow-sm">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-primary" />
            </span>
            {t.hero.badge}
          </div>

          {/* Headline */}
          <h1 className="font-display text-balance text-4xl sm:text-5xl md:text-6xl lg:text-7xl font-bold tracking-tight text-foreground leading-[1.1] mb-6 animate-in fade-in-0 slide-in-from-bottom-3 duration-700">
            {t.hero.title1}
            <span className="block mt-2 bg-gradient-to-r from-primary via-chart-2 to-primary bg-clip-text text-transparent">
              {t.hero.title2}
            </span>
          </h1>

          <p className="text-pretty text-base sm:text-lg md:text-xl text-muted-foreground max-w-2xl mx-auto mb-10 leading-relaxed px-4 animate-in fade-in-0 slide-in-from-bottom-3 duration-700 [animation-delay:120ms] fill-mode-both">
            {t.hero.desc}
          </p>

          {/* CTA */}
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-16">
            <Link href="/dashboard">
              <Button size="lg" className="btn-gradient text-primary-foreground shadow-lg hover:shadow-xl transition-all px-8 h-12 text-base gap-2 w-full sm:w-auto">
                {t.hero.start}
                <ArrowRight className="w-4 h-4" />
              </Button>
            </Link>
            <Link href="#pricing">
              <Button variant="outline" size="lg" className="h-12 px-8 text-base border-border hover:border-primary/30 hover:bg-accent w-full sm:w-auto">
                {t.hero.plans}
              </Button>
            </Link>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 max-w-3xl mx-auto">
            {stats.map((s, i) => {
              const Icon = s.icon
              return (
                <div
                  key={s.label}
                  className="bg-card rounded-2xl p-5 sm:p-6 card-shadow border border-border/50 hover:card-shadow-hover transition-shadow animate-in fade-in-0 slide-in-from-bottom-3 duration-500 fill-mode-both"
                  style={{ animationDelay: `${300 + i * 80}ms` }}
                >
                  <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center mx-auto mb-3">
                    <Icon className="w-5 h-5 text-primary" />
                  </div>
                  <div className="font-display text-2xl sm:text-3xl font-bold text-foreground mb-1 tabular-nums">{s.value}</div>
                  <div className="text-sm text-muted-foreground">{s.label}</div>
                </div>
              )
            })}
          </div>
        </div>
      </div>
    </section>
  )
}
