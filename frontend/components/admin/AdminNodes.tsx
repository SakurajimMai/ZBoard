"use client"

import { useState } from "react"
import { Plus, Edit, Trash2, RefreshCw, ToggleLeft, ToggleRight } from "lucide-react"
import { Button } from "@/components/ui/button"

const nodes = [
  { id: 1, name: "香港 IPLC 01", host: "hk1.zboard.io", port: 443, protocol: "Vmess", type: "IPLC", latency: 8, load: 32, status: "online" },
  { id: 2, name: "香港 IPLC 02", host: "hk2.zboard.io", port: 443, protocol: "Vless", type: "IPLC", latency: 10, load: 45, status: "online" },
  { id: 3, name: "日本 BGP 01", host: "jp1.zboard.io", port: 8443, protocol: "Trojan", type: "BGP", latency: 15, load: 28, status: "online" },
  { id: 4, name: "新加坡 01", host: "sg1.zboard.io", port: 443, protocol: "Vmess", type: "BGP", latency: 22, load: 40, status: "online" },
  { id: 5, name: "美国 西岸 01", host: "us-w1.zboard.io", port: 443, protocol: "Trojan", type: "BGP", latency: 138, load: 20, status: "online" },
  { id: 6, name: "荷兰 阿姆斯特丹", host: "nl1.zboard.io", port: 8443, protocol: "Vless", type: "BGP", latency: 190, load: 0, status: "maintenance" },
  { id: 7, name: "德国 法兰克福", host: "de1.zboard.io", port: 443, protocol: "Vless", type: "BGP", latency: 195, load: 22, status: "online" },
]

function getLoadColor(load: number) {
  if (load < 40) return "bg-green-500"
  if (load < 70) return "bg-yellow-500"
  return "bg-red-500"
}

export default function AdminNodes() {
  const [nodeList, setNodeList] = useState(nodes)

  const toggleStatus = (id: number) => {
    setNodeList((prev) =>
      prev.map((n) =>
        n.id === id
          ? { ...n, status: n.status === "online" ? "maintenance" : "online" }
          : n
      )
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">节点管理</h1>
          <p className="text-sm text-muted-foreground mt-1">
            共 {nodeList.length} 个节点 ·{" "}
            <span className="text-green-400">{nodeList.filter((n) => n.status === "online").length} 在线</span>
            {" · "}
            <span className="text-yellow-400">{nodeList.filter((n) => n.status === "maintenance").length} 维护</span>
          </p>
        </div>
        <div className="flex gap-3">
          <Button variant="outline" className="gap-2 hover:border-primary/50">
            <RefreshCw className="w-4 h-4" />
            测速所有节点
          </Button>
          <Button className="gap-2 bg-primary text-primary-foreground hover:bg-primary/90">
            <Plus className="w-4 h-4" />
            添加节点
          </Button>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-secondary/50 border-b border-border">
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">节点名称</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">地址</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">协议</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">线路</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">延迟</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">负载</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">状态</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              {nodeList.map((n) => (
                <tr key={n.id} className="border-b border-border/50 hover:bg-accent/50 transition-colors">
                  <td className="px-5 py-3 font-medium text-foreground">{n.name}</td>
                  <td className="px-5 py-3">
                    <span className="font-mono text-xs text-muted-foreground">{n.host}:{n.port}</span>
                  </td>
                  <td className="px-5 py-3">
                    <span className="text-xs rounded bg-primary/10 text-primary px-2 py-0.5 font-mono">{n.protocol}</span>
                  </td>
                  <td className="px-5 py-3">
                    <span className={`text-xs rounded px-2 py-0.5 ${
                      n.type === "IPLC" ? "bg-primary/20 text-primary" : "bg-secondary text-muted-foreground"
                    }`}>
                      {n.type}
                    </span>
                  </td>
                  <td className="px-5 py-3">
                    <span className="font-mono text-xs text-primary">{n.status === "maintenance" ? "-" : `${n.latency}ms`}</span>
                  </td>
                  <td className="px-5 py-3">
                    {n.status !== "maintenance" ? (
                      <div className="flex items-center gap-2">
                        <div className="w-16 h-1.5 rounded-full bg-secondary overflow-hidden">
                          <div className={`h-full rounded-full ${getLoadColor(n.load)}`} style={{ width: `${n.load}%` }} />
                        </div>
                        <span className="text-xs text-muted-foreground">{n.load}%</span>
                      </div>
                    ) : (
                      <span className="text-xs text-muted-foreground">-</span>
                    )}
                  </td>
                  <td className="px-5 py-3">
                    <span className={`inline-flex items-center gap-1.5 text-xs ${
                      n.status === "online" ? "text-green-400" : "text-yellow-400"
                    }`}>
                      <span className={`w-1.5 h-1.5 rounded-full ${n.status === "online" ? "bg-green-500" : "bg-yellow-500"}`} />
                      {n.status === "online" ? "在线" : "维护中"}
                    </span>
                  </td>
                  <td className="px-5 py-3">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => toggleStatus(n.id)}
                        className="hover:text-primary transition-colors"
                        aria-label="切换状态"
                      >
                        {n.status === "online" ? (
                          <ToggleRight className="w-5 h-5 text-primary" />
                        ) : (
                          <ToggleLeft className="w-5 h-5 text-muted-foreground" />
                        )}
                      </button>
                      <button className="p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors">
                        <Edit className="w-3.5 h-3.5" />
                      </button>
                      <button className="p-1.5 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors">
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
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
