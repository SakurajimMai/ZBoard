"use client"

import { useEffect, useState } from "react"
import { Globe, Server } from "lucide-react"
import { getPlans } from "@/lib/api"

// This component shows available regions on the landing page.
// Since the public API doesn't expose node details (security), we show
// a simple "X regions available" message based on whether plans exist.
export default function Nodes() {
  const [hasPlans, setHasPlans] = useState(false)

  useEffect(() => {
    getPlans()
      .then((res) => setHasPlans((res.items || []).length > 0))
      .catch(() => {})
  }, [])

  return (
    <section id="nodes" className="py-20 px-4 bg-muted/30">
      <div className="max-w-6xl mx-auto text-center">
        <h2 className="text-3xl font-bold">全球节点覆盖</h2>
        <p className="text-muted-foreground mt-2 max-w-xl mx-auto">
          多地区高速节点，支持 VLESS+Reality、Hysteria2、TUIC 等抗封锁协议，智能选路确保最佳连接体验。
        </p>

        <div className="mt-12 grid grid-cols-1 sm:grid-cols-3 gap-6 max-w-2xl mx-auto">
          <div className="rounded-2xl border bg-card p-6 text-center">
            <Globe className="w-8 h-8 text-primary mx-auto mb-3" />
            <div className="text-2xl font-bold">多地区</div>
            <p className="text-sm text-muted-foreground mt-1">全球节点覆盖</p>
          </div>
          <div className="rounded-2xl border bg-card p-6 text-center">
            <Server className="w-8 h-8 text-primary mx-auto mb-3" />
            <div className="text-2xl font-bold">高可用</div>
            <p className="text-sm text-muted-foreground mt-1">自动故障转移</p>
          </div>
          <div className="rounded-2xl border bg-card p-6 text-center">
            <div className="w-8 h-8 rounded-full bg-green-100 flex items-center justify-center mx-auto mb-3">
              <div className="w-3 h-3 rounded-full bg-green-500" />
            </div>
            <div className="text-2xl font-bold">{hasPlans ? "在线" : "—"}</div>
            <p className="text-sm text-muted-foreground mt-1">服务状态</p>
          </div>
        </div>

        <p className="mt-8 text-sm text-muted-foreground">
          注册后可在控制台查看完整节点列表和实时状态。
        </p>
      </div>
    </section>
  )
}
