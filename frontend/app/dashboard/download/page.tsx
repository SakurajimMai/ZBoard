import { Download, ExternalLink, Monitor, Apple, Smartphone, TabletSmartphone, Bot, Wrench, Radio, Sparkles, Star } from "lucide-react"
import { Button } from "@/components/ui/button"

const clients = [
  { name: "Clash for Windows", os: "Windows", version: "0.20.39", icon: Monitor, note: "推荐，功能最全" },
  { name: "ClashX Pro", os: "macOS", version: "1.3.8", icon: Apple, note: "macOS 原生体验" },
  { name: "Shadowrocket", os: "iOS", version: "2.2.52", icon: Smartphone, note: "需 App Store 购买" },
  { name: "Quantumult X", os: "iOS", version: "1.0.30", icon: TabletSmartphone, note: "高级规则支持" },
  { name: "Clash for Android", os: "Android", version: "2.5.12", icon: Bot, note: "免费开源" },
  { name: "NekoBox", os: "Android", version: "1.2.9", icon: Wrench, note: "支持多核心" },
  { name: "OpenClash", os: "OpenWrt", version: "0.46.002", icon: Radio, note: "路由器透明代理" },
  { name: "Stash", os: "iOS/macOS", version: "2.4.1", icon: Sparkles, note: "界面精美" },
]

export default function DownloadPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">客户端下载</h1>
        <p className="text-sm text-muted-foreground mt-1">下载适合您设备的代理客户端，导入订阅链接即可使用。</p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {clients.map((c) => {
          const Icon = c.icon
          return (
            <div key={c.name} className="rounded-xl border border-border bg-card p-5 flex items-center gap-4 hover:border-primary/30 transition-colors card-shadow">
              <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center flex-shrink-0">
                <Icon className="w-6 h-6 text-primary" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="font-medium text-foreground">{c.name}</p>
                  {c.note.includes("推荐") && (
                    <span className="inline-flex items-center gap-1 text-xs rounded-full bg-primary/10 text-primary px-2 py-0.5">
                      <Star className="w-3 h-3" />
                      推荐
                    </span>
                  )}
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">{c.os} · v{c.version} · {c.note}</p>
              </div>
              <Button size="sm" variant="outline" className="gap-1.5 flex-shrink-0 hover:border-primary/50">
                <Download className="w-3.5 h-3.5" />
                下载
              </Button>
            </div>
          )
        })}
      </div>

      <div className="rounded-xl border border-border bg-card p-6 card-shadow">
        <h2 className="font-semibold text-foreground mb-4">使用教程</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
          {[
            { title: "Windows 使用教程", link: "#" },
            { title: "macOS 使用教程", link: "#" },
            { title: "iOS 使用教程", link: "#" },
            { title: "Android 使用教程", link: "#" },
            { title: "路由器配置教程", link: "#" },
            { title: "常见问题排查", link: "#" },
          ].map((t) => (
            <a
              key={t.title}
              href={t.link}
              className="flex items-center gap-2 text-muted-foreground hover:text-primary transition-colors"
            >
              <ExternalLink className="w-3.5 h-3.5 flex-shrink-0" />
              {t.title}
            </a>
          ))}
        </div>
      </div>
    </div>
  )
}
