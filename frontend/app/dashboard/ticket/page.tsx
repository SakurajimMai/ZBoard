"use client"

import { MessageSquare } from "lucide-react"

export default function TicketPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">工单支持</h1>
        <p className="text-sm text-muted-foreground mt-1">提交问题或反馈</p>
      </div>

      <div className="flex flex-col items-center justify-center py-20 text-center">
        <div className="w-16 h-16 rounded-2xl bg-muted flex items-center justify-center mb-4">
          <MessageSquare className="w-8 h-8 text-muted-foreground" />
        </div>
        <h3 className="text-lg font-medium">功能开发中</h3>
        <p className="text-sm text-muted-foreground mt-2 max-w-sm">
          工单系统正在开发中，即将上线。届时您可以通过此页面提交问题和查看回复。
        </p>
      </div>
    </div>
  )
}
