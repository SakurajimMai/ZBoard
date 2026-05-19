"use client"

import { useState } from "react"
import { Search, Star, Globe, Wifi, Signal } from "lucide-react"
import { Input } from "@/components/ui/input"

const nodes = [
  { name: "香港 IPLC 01", region: "香港", flag: "HK", latency: 8, load: 32, protocol: "Vmess", type: "IPLC", status: "online" },
  { name: "香港 IPLC 02", region: "香港", flag: "HK", latency: 10, load: 45, protocol: "Vless", type: "IPLC", status: "online" },
  { name: "日本 BGP 01", region: "日本", flag: "JP", latency: 15, load: 28, protocol: "Trojan", type: "BGP", status: "online" },
  { name: "日本 BGP 02", region: "日本", flag: "JP", latency: 18, load: 61, protocol: "Vmess", type: "BGP", status: "online" },
  { name: "新加坡 01", region: "新加坡", flag: "SG", latency: 22, load: 40, protocol: "Vmess", type: "BGP", status: "online" },
  { name: "台湾 BGP 01", region: "台湾", flag: "TW", latency: 12, load: 55, protocol: "Vless", type: "BGP", status: "online" },
  { name: "美国 西岸 01", region: "美国", flag: "US", latency: 138, load: 20, protocol: "Trojan", type: "BGP", status: "online" },
  { name: "美国 东岸 01", region: "美国", flag: "US", latency: 158, load: 35, protocol: "Vmess", type: "BGP", status: "online" },
  { name: "英国 伦敦 01", region: "英国", flag: "UK", latency: 185, load: 18, protocol: "Trojan", type: "BGP", status: "online" },
  { name: "德国 法兰克福", region: "德国", flag: "DE", latency: 195, load: 22, protocol: "Vless", type: "BGP", status: "online" },
  { name: "韩国 首尔 01", region: "韩国", flag: "KR", latency: 18, load: 70, protocol: "Vmess", type: "BGP", status: "online" },
  { name: "荷兰 阿姆斯特丹", region: "荷兰", flag: "NL", latency: 190, load: 0, protocol: "Vless", type: "BGP", status: "maintenance" },
]

const regions = ["全部", "香港", "日本", "新加坡", "台湾", "美国", "英国", "德国", "韩国"]

function getLoadColor(load: number) {
  if (load < 40) return "bg-success"
  if (load < 70) return "bg-yellow-500"
  return "bg-destructive"
}

export default function Subscription() {
  const [search, setSearch] = useState("")
  const [region, setRegion] = useState("全部")
  const [favorites, setFavorites] = useState<string[]>(["香港 IPLC 01"])

  const filtered = nodes.filter((n) => {
    const matchSearch = n.name.toLowerCase().includes(search.toLowerCase()) || n.region.includes(search)
    const matchRegion = region === "全部" || n.region === region
    return matchSearch && matchRegion
  })

  const toggleFav = (name: string) => {
    setFavorites((prev) => prev.includes(name) ? prev.filter((f) => f !== name) : [...prev, name])
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl sm:text-3xl font-bold text-foreground">我的订阅</h1>
        <p className="text-sm text-muted-foreground mt-1">选择并查看可用节点，点击收藏常用节点。</p>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4">
        <div className="relative">
          <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="搜索节点名称或地区..."
            className="pl-11 bg-card border-border/50 h-11 rounded-xl"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="flex gap-2 flex-wrap">
          {regions.map((r) => (
            <button
              key={r}
              onClick={() => setRegion(r)}
              className={`px-4 py-2 rounded-xl text-sm font-medium transition-all ${
                region === r
                  ? "bg-primary text-primary-foreground shadow-sm"
                  : "bg-card text-muted-foreground hover:text-foreground border border-border/50 hover:border-primary/30"
              }`}
            >
              {r}
            </button>
          ))}
        </div>
      </div>

      {/* Nodes grid (mobile) / table (desktop) */}
      <div className="block sm:hidden space-y-3">
        {filtered.map((node) => (
          <div key={node.name} className="rounded-2xl border border-border/50 bg-card p-4 card-shadow">
            <div className="flex items-start justify-between mb-3">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center text-xs font-bold text-primary">
                  {node.flag}
                </div>
                <div>
                  <p className="font-semibold text-foreground">{node.name}</p>
                  <p className="text-xs text-muted-foreground">{node.type} · {node.protocol}</p>
                </div>
              </div>
              <button
                onClick={() => toggleFav(node.name)}
                className="hover:text-yellow-500 transition-colors"
                aria-label={favorites.includes(node.name) ? "取消收藏" : "收藏节点"}
              >
                <Star
                  className={`w-5 h-5 ${favorites.includes(node.name) ? "text-yellow-500 fill-yellow-500" : "text-muted-foreground"}`}
                />
              </button>
            </div>
            <div className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-4">
                <span className={`font-mono ${node.status === "maintenance" ? "text-muted-foreground" : "text-primary"}`}>
                  {node.status === "maintenance" ? "-" : `${node.latency}ms`}
                </span>
                {node.status !== "maintenance" && (
                  <div className="flex items-center gap-2">
                    <div className="w-12 h-1.5 rounded-full bg-muted overflow-hidden">
                      <div
                        className={`h-full rounded-full ${getLoadColor(node.load)}`}
                        style={{ width: `${node.load}%` }}
                      />
                    </div>
                    <span className="text-xs text-muted-foreground">{node.load}%</span>
                  </div>
                )}
              </div>
              <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
                node.status === "online" ? "text-success" : "text-yellow-500"
              }`}>
                <span className={`w-1.5 h-1.5 rounded-full ${
                  node.status === "online" ? "bg-success" : "bg-yellow-500"
                }`} />
                {node.status === "online" ? "在线" : "维护中"}
              </span>
            </div>
          </div>
        ))}
      </div>

      {/* Desktop table */}
      <div className="hidden sm:block rounded-2xl border border-border/50 bg-card overflow-hidden card-shadow">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/50">
                <th className="text-left px-5 py-4 text-muted-foreground font-medium">节点名称</th>
                <th className="text-left px-5 py-4 text-muted-foreground font-medium">协议</th>
                <th className="text-left px-5 py-4 text-muted-foreground font-medium">延迟</th>
                <th className="text-left px-5 py-4 text-muted-foreground font-medium">负载</th>
                <th className="text-left px-5 py-4 text-muted-foreground font-medium">状态</th>
                <th className="text-left px-5 py-4 text-muted-foreground font-medium">收藏</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((node) => (
                <tr key={node.name} className="border-b border-border/50 hover:bg-accent/50 transition-colors">
                  <td className="px-5 py-4">
                    <div className="flex items-center gap-3">
                      <div className="w-9 h-9 rounded-xl bg-primary/10 flex items-center justify-center text-xs font-bold text-primary">
                        {node.flag}
                      </div>
                      <div>
                        <p className="font-medium text-foreground">{node.name}</p>
                        <p className="text-xs text-muted-foreground">{node.type}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-5 py-4">
                    <span className="px-2.5 py-1 rounded-lg text-xs bg-primary/10 text-primary font-mono font-medium">{node.protocol}</span>
                  </td>
                  <td className="px-5 py-4">
                    <span className={`font-mono text-sm font-medium ${node.status === "maintenance" ? "text-muted-foreground" : "text-primary"}`}>
                      {node.status === "maintenance" ? "-" : `${node.latency}ms`}
                    </span>
                  </td>
                  <td className="px-5 py-4">
                    {node.status !== "maintenance" ? (
                      <div className="flex items-center gap-2">
                        <div className="w-16 h-2 rounded-full bg-muted overflow-hidden">
                          <div
                            className={`h-full rounded-full ${getLoadColor(node.load)}`}
                            style={{ width: `${node.load}%` }}
                          />
                        </div>
                        <span className="text-xs text-muted-foreground">{node.load}%</span>
                      </div>
                    ) : (
                      <span className="text-xs text-muted-foreground">-</span>
                    )}
                  </td>
                  <td className="px-5 py-4">
                    <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
                      node.status === "online" ? "text-success" : "text-yellow-500"
                    }`}>
                      <span className={`w-2 h-2 rounded-full ${
                        node.status === "online" ? "bg-success" : "bg-yellow-500"
                      }`} />
                      {node.status === "online" ? "在线" : "维护中"}
                    </span>
                  </td>
                  <td className="px-5 py-4">
                    <button
                      onClick={() => toggleFav(node.name)}
                      className="hover:text-yellow-500 transition-colors"
                      aria-label={favorites.includes(node.name) ? "取消收藏" : "收藏节点"}
                    >
                      <Star
                        className={`w-5 h-5 ${favorites.includes(node.name) ? "text-yellow-500 fill-yellow-500" : "text-muted-foreground"}`}
                      />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="px-5 py-4 text-sm text-muted-foreground border-t border-border">
          共 {filtered.length} 个节点
        </div>
      </div>

      {/* Quick info */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="rounded-2xl border border-border/50 bg-card p-5 sm:p-6 card-shadow">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
              <Globe className="w-5 h-5 text-primary" />
            </div>
            <h3 className="font-semibold text-foreground">订阅导入指引</h3>
          </div>
          <ol className="space-y-3 text-sm text-muted-foreground list-decimal list-inside">
            <li>在控制台页面复制您的订阅链接</li>
            <li>打开 Clash / V2rayN 等客户端</li>
            <li>添加订阅 → 粘贴链接 → 更新</li>
            <li>选择节点，开启系统代理即可</li>
          </ol>
        </div>
        <div className="rounded-2xl border border-border/50 bg-card p-5 sm:p-6 card-shadow">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
              <Signal className="w-5 h-5 text-primary" />
            </div>
            <h3 className="font-semibold text-foreground">线路说明</h3>
          </div>
          <ul className="space-y-3 text-sm text-muted-foreground">
            <li><span className="text-primary font-semibold">IPLC</span> — 专线直连，延迟最低，稳定性最强</li>
            <li><span className="text-primary font-semibold">BGP</span> — 多运营商优化，适合日常使用</li>
            <li>负载 {'<'} 40% 绿色，40-70% 黄色，{'>'}70% 红色</li>
          </ul>
        </div>
      </div>
    </div>
  )
}
