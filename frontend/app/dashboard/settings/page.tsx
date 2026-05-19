"use client"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">账户设置</h1>
        <p className="text-sm text-muted-foreground mt-1">管理您的个人信息和安全设置。</p>
      </div>

      <div className="rounded-xl border border-border bg-card p-6 space-y-5">
        <h2 className="font-semibold text-foreground">基本信息</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="text-sm text-muted-foreground mb-1.5 block">邮箱地址</label>
            <Input defaultValue="user@example.com" className="bg-secondary border-border" />
          </div>
          <div>
            <label className="text-sm text-muted-foreground mb-1.5 block">用户名</label>
            <Input defaultValue="myuser" className="bg-secondary border-border" />
          </div>
        </div>
        <Button className="bg-primary text-primary-foreground hover:bg-primary/90">保存更改</Button>
      </div>

      <div className="rounded-xl border border-border bg-card p-6 space-y-5">
        <h2 className="font-semibold text-foreground">修改密码</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="text-sm text-muted-foreground mb-1.5 block">当前密码</label>
            <Input type="password" placeholder="••••••••" className="bg-secondary border-border" />
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="text-sm text-muted-foreground mb-1.5 block">新密码</label>
            <Input type="password" placeholder="••••••••" className="bg-secondary border-border" />
          </div>
          <div>
            <label className="text-sm text-muted-foreground mb-1.5 block">确认新密码</label>
            <Input type="password" placeholder="••••••••" className="bg-secondary border-border" />
          </div>
        </div>
        <Button variant="outline" className="hover:border-primary/50">更新密码</Button>
      </div>

      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-6">
        <h2 className="font-semibold text-foreground mb-2">危险区域</h2>
        <p className="text-sm text-muted-foreground mb-4">注销账户后，所有数据将被永久删除且无法恢复。</p>
        <Button variant="outline" className="border-destructive/50 text-destructive hover:bg-destructive/10 hover:text-destructive">
          注销账户
        </Button>
      </div>
    </div>
  )
}
