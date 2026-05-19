"use client"

import { Globe, Wifi, Server } from "lucide-react"
import { useI18n } from "@/lib/i18n/context"

const regions = [
  { code: "HK", latency: "8ms",   count: 20, status: "online" },
  { code: "JP", latency: "15ms",  count: 18, status: "online" },
  { code: "SG", latency: "22ms",  count: 16, status: "online" },
  { code: "TW", latency: "12ms",  count: 14, status: "online" },
  { code: "US", latency: "140ms", count: 30, status: "online" },
  { code: "UK", latency: "185ms", count: 12, status: "online" },
  { code: "DE", latency: "195ms", count: 10, status: "online" },
  { code: "KR", latency: "18ms",  count: 15, status: "online" },
  { code: "NL", latency: "190ms", count: 8,  status: "maintenance" },
  { code: "CA", latency: "155ms", count: 10, status: "online" },
  { code: "AU", latency: "88ms",  count: 8,  status: "online" },
  { code: "IN", latency: "75ms",  count: 6,  status: "online" },
]

// Country names per locale are handled via browser-native Intl API
function getCountryName(code: string, locale: string) {
  try {
    return new Intl.DisplayNames([locale], { type: "region" }).of(code) ?? code
  } catch {
    return code
  }
}

export default function Nodes() {
  const { t, locale } = useI18n()

  return (
    <section id="nodes" className="py-20 sm:py-28 px-4 sm:px-6 lg:px-8 bg-muted/30">
      <div className="mx-auto max-w-7xl">
        <div className="text-center mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/5 px-4 py-2 text-sm text-primary font-medium mb-4">
            {t.nodes.title}
          </div>
          <h2 className="text-balance text-3xl sm:text-4xl md:text-5xl font-bold text-foreground mb-4">
            {t.nodes.subtitle}
          </h2>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-3 sm:gap-4">
          {regions.map((r) => (
            <div
              key={r.code}
              className="rounded-2xl border border-border/50 bg-card p-4 sm:p-5 card-shadow hover:card-shadow-hover hover:border-primary/20 transition-all group"
            >
              <div className="flex items-center justify-between mb-3">
                <span className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center text-xs font-bold text-primary group-hover:bg-primary/15 transition-colors">
                  {r.code}
                </span>
                <span className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${r.status === "online" ? "bg-success" : "bg-yellow-500"}`} />
              </div>
              <div className="font-semibold text-sm text-foreground mb-1 truncate">
                {getCountryName(r.code, locale)}
              </div>
              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>{r.count} {t.nodes.online === "Online" ? "nodes" : "节点"}</span>
                <span className="text-primary font-mono font-medium">{r.latency}</span>
              </div>
            </div>
          ))}
        </div>

        {/* Line types */}
        <div className="mt-12 grid grid-cols-1 md:grid-cols-3 gap-4 sm:gap-5">
          {[
            { icon: Globe, title: "IPLC", desc: "国际私人租用电路，点对点直连，延迟最低。" },
            { icon: Wifi,  title: "BGP",  desc: "自动选择最优出口，多运营商互联，智能规避拥堵。" },
            { icon: Server, title: "中转加速", desc: "国内中转 + 海外出口，有效绕过跨境访问障碍。" },
          ].map((item) => {
            const Icon = item.icon
            return (
              <div key={item.title} className="rounded-2xl border border-border/50 bg-card p-6 card-shadow">
                <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mb-4">
                  <Icon className="w-6 h-6 text-primary" />
                </div>
                <h3 className="font-semibold text-foreground text-lg mb-2">{item.title}</h3>
                <p className="text-sm text-muted-foreground leading-relaxed">{item.desc}</p>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}
