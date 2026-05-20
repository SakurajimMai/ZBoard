"use client"

import { MessageSquare } from "lucide-react"

export default function AdminTicketsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">工单管理</h1>
        <p className="text-sm text-muted-foreground mt-1">管理用户提交的工单</p>
      </div>

      <div className="flex flex-col items-center justify-center py-20 text-center">
        <div className="w-16 h-16 rounded-2xl bg-muted flex items-center justify-center mb-4">
          <MessageSquare className="w-8 h-8 text-muted-foreground" />
        </div>
        <h3 className="text-lg font-medium">功能开发中</h3>
        <p className="text-sm text-muted-foreground mt-2 max-w-sm">
          工单系统正在开发中，即将上线。届时用户可以通过工单系统提交问题和反馈。
        </p>
      </div>
    </div>
  )
}
