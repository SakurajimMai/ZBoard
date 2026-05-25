"use client"

import { FormEvent, ReactNode, useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { CircleUserRound, KeyRound, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { changeMyPassword, deleteMyAccount, getMe, logout } from "@/lib/api"

type UserProfile = {
  email?: string
  status?: string
}

export default function SettingsPage() {
  const router = useRouter()
  const [user, setUser] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [passwordForm, setPasswordForm] = useState({
    currentPassword: "",
    newPassword: "",
    confirmPassword: "",
  })
  const [deletePassword, setDeletePassword] = useState("")
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const [savingPassword, setSavingPassword] = useState(false)
  const [deletingAccount, setDeletingAccount] = useState(false)

  useEffect(() => {
    getMe()
      .then((res) => setUser(res.user))
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  const clearNotice = () => {
    setMessage("")
    setError("")
  }

  const handlePasswordSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    clearNotice()

    if (passwordForm.newPassword.length < 6) {
      setError("新密码至少需要 6 位")
      return
    }
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setError("两次输入的新密码不一致")
      return
    }

    setSavingPassword(true)
    try {
      await changeMyPassword(passwordForm.currentPassword, passwordForm.newPassword)
      setPasswordForm({ currentPassword: "", newPassword: "", confirmPassword: "" })
      setMessage("密码已更新")
    } catch (err: any) {
      setError(err.message || "更新密码失败")
    } finally {
      setSavingPassword(false)
    }
  }

  const handleDeleteAccount = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    clearNotice()

    if (!deletePassword) {
      setError("请输入当前密码后再注销账户")
      return
    }
    if (!window.confirm("确认注销账户？注销后所有会话会失效，账户将无法继续使用。")) {
      return
    }

    setDeletingAccount(true)
    try {
      await deleteMyAccount(deletePassword)
      logout()
      router.replace("/login")
    } catch (err: any) {
      setError(err.message || "注销账户失败")
      setDeletingAccount(false)
    }
  }

  if (loading) {
    return <div className="p-8 text-muted-foreground">加载中...</div>
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">账户设置</h1>
        <p className="mt-1 text-sm text-muted-foreground">管理个人账户信息和安全设置。</p>
      </div>

      {(message || error) && (
        <div
          className={`rounded-lg border px-4 py-3 text-sm ${
            error
              ? "border-destructive/30 bg-destructive/10 text-destructive"
              : "border-emerald-500/30 bg-emerald-500/10 text-emerald-700"
          }`}
        >
          {error || message}
        </div>
      )}

      <section className="rounded-xl border border-border bg-card p-6">
        <div className="mb-5 flex items-center gap-4">
          <CircleUserRound className="h-12 w-12 flex-shrink-0 text-foreground/80" strokeWidth={1.8} />
          <div>
            <h2 className="font-semibold text-foreground">基本信息</h2>
            <p className="text-sm text-muted-foreground">当前登录账户</p>
          </div>
        </div>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Field label="邮箱地址">
            <Input value={user?.email || ""} readOnly className="bg-secondary border-border" />
          </Field>
          <Field label="账户状态">
            <Input value={user?.status === "active" ? "正常" : user?.status || "-"} readOnly className="bg-secondary border-border" />
          </Field>
        </div>
      </section>

      <section className="rounded-xl border border-border bg-card p-6">
        <div className="mb-5 flex items-center gap-3">
          <KeyRound className="h-5 w-5 text-primary" />
          <h2 className="font-semibold text-foreground">修改密码</h2>
        </div>
        <form onSubmit={handlePasswordSubmit} className="space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <Field label="当前密码">
              <Input
                type="password"
                value={passwordForm.currentPassword}
                onChange={(e) => setPasswordForm((current) => ({ ...current, currentPassword: e.target.value }))}
                autoComplete="current-password"
                required
              />
            </Field>
            <Field label="新密码">
              <Input
                type="password"
                value={passwordForm.newPassword}
                onChange={(e) => setPasswordForm((current) => ({ ...current, newPassword: e.target.value }))}
                autoComplete="new-password"
                required
              />
            </Field>
            <Field label="确认新密码">
              <Input
                type="password"
                value={passwordForm.confirmPassword}
                onChange={(e) => setPasswordForm((current) => ({ ...current, confirmPassword: e.target.value }))}
                autoComplete="new-password"
                required
              />
            </Field>
          </div>
          <Button type="submit" disabled={savingPassword}>
            {savingPassword ? "更新中..." : "更新密码"}
          </Button>
        </form>
      </section>

      <section className="rounded-xl border border-destructive/40 bg-destructive/5 p-6">
        <div className="mb-5 flex items-center gap-3">
          <Trash2 className="h-5 w-5 text-destructive" />
          <div>
            <h2 className="font-semibold text-foreground">危险区域</h2>
            <p className="text-sm text-muted-foreground">注销账户后，所有会话会失效，节点访问也会被停用。</p>
          </div>
        </div>
        <form onSubmit={handleDeleteAccount} className="space-y-4">
          <Field label="当前密码">
            <Input
              type="password"
              value={deletePassword}
              onChange={(e) => setDeletePassword(e.target.value)}
              autoComplete="current-password"
              required
            />
          </Field>
          <Button type="submit" variant="destructive" disabled={deletingAccount}>
            {deletingAccount ? "注销中..." : "注销账户"}
          </Button>
        </form>
      </section>
    </div>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}
