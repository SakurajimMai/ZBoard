"use client"

import { useEffect, useState } from "react"
import { Plus, RefreshCw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { adminGetNodes, adminSyncNodeConfig } from "@/lib/api"

export default function AdminNodes() {
  const [nodes, setNodes] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  const load = () => {
    setLoading(true)
    adminGetNodes()
      .then((res) => setNodes(res.items || []))
      .catch(console.error)
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [])

  const handleSync = async (nodeId: number) => {
    try {
      const res = await adminSyncNodeConfig(nodeId)
      alert(`配置同步任务已创建: ${res.task_id}`)
    } catch (err: any) {
      alert(err.message || "同步失败")
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">节点管理</h1>
          <p className="text-sm text-muted-foreground mt-1">共 {nodes.length} 个节点</p>
        </div>
        <Button size="sm" onClick={load} variant="outline">
          <RefreshCw className="w-4 h-4 mr-1" /> 刷新
        </Button>
      </div>

      {nodes.length === 0 ? (
        <div className="text-center text-muted-foreground py-12">暂无节点，请通过 API 创建</div>
      ) : (
        <div className="rounded-xl border bg-card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted/50 border-b">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ID</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">名称</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden md:table-cell">地区</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">协议</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground hidden lg:table-cell">地址</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">状态</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">操作</th>
                </tr>
              </thead>
              <tbody>
                {nodes.map((n: any) => (
                  <tr key={n.id} className="border-b hover:bg-accent/50">
                    <td className="px-4 py-3">{n.id}</td>
                    <td className="px-4 py-3 font-medium">{n.name}</td>
                    <td className="px-4 py-3 hidden md:table-cell">{n.region || "-"}</td>
                    <td className="px-4 py-3 hidden lg:table-cell">
                      <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded">{n.protocol}</span>
                    </td>
                    <td className="px-4 py-3 hidden lg:table-cell text-muted-foreground">{n.host}:{n.port}</td>
                    <td className="px-4 py-3">
                      <span className={`text-xs rounded-full px-2 py-1 ${
                        n.status === "active" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-600"
                      }`}>
                        {n.status}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <Button size="sm" variant="ghost" onClick={() => handleSync(n.id)}>
                        <RefreshCw className="w-3 h-3 mr-1" /> 同步
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
