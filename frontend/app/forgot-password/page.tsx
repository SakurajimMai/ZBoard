"use client"

import { useCallback, useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { resetPassword, sendEmailCode, getPublicSettings, ApiError } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Captcha, captchaEnabled } from "@/components/captcha"

const COOLDOWN_SECONDS = 120

export default function ForgotPasswordPage() {
  const router = useRouter()
  const [email, setEmail] = useState("")
  const [code, setCode] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [settings, setSettings] = useState<Record<string, string>>({})
  const [captchaToken, setCaptchaToken] = useState("")
  const [captchaKey, setCaptchaKey] = useState(0)
  const [cooldown, setCooldown] = useState(0)
  const [sending, setSending] = useState(false)
  const [error, setError] = useState("")
  const [hint, setHint] = useState("")
  const [done, setDone] = useState(false)
  const [loading, setLoading] = useState(false)
  const timerRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    getPublicSettings()
      .then((res) => setSettings(res.settings || {}))
      .catch(() => {})
    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
    }
  }, [])

  const needCaptcha = captchaEnabled(settings, "forgot")
  const handleToken = useCallback((t: string) => setCaptchaToken(t), [])
  const resetCaptcha = useCallback(() => {
    setCaptchaToken("")
    setCaptchaKey((v) => v + 1)
  }, [])

  const startCooldown = () => {
    setCooldown(COOLDOWN_SECONDS)
    if (timerRef.current) clearInterval(timerRef.current)
    timerRef.current = setInterval(() => {
      setCooldown((prev) => {
        if (prev <= 1) {
          if (timerRef.current) clearInterval(timerRef.current)
          return 0
        }
        return prev - 1
      })
    }, 1000)
  }

  const handleSendCode = async () => {
    setError("")
    setHint("")
    if (!email.trim()) {
      setError("请先填写邮箱")
      return
    }
    if (needCaptcha && !captchaToken) {
      setError("请先完成人机验证")
      return
    }
    setSending(true)
    try {
      await sendEmailCode(email.trim(), "reset_password", captchaToken)
      setHint("验证码已发送，请检查邮箱（含垃圾箱）")
      startCooldown()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "验证码发送失败")
    } finally {
      if (needCaptcha) resetCaptcha()
      setSending(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    setHint("")
    if (newPassword !== confirmPassword) {
      setError("两次输入的密码不一致")
      return
    }
    if (newPassword.length < 6) {
      setError("密码至少 6 位")
      return
    }
    if (!code.trim()) {
      setError("请填写邮箱验证码")
      return
    }
    if (needCaptcha && !captchaToken) {
      setError("请先完成人机验证")
      return
    }
    setLoading(true)
    let success = false
    try {
      await resetPassword(email.trim(), newPassword, code.trim(), captchaToken)
      success = true
      setDone(true)
      setHint("密码已重置，3 秒后跳转到登录页")
      setTimeout(() => router.push("/login"), 3000)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "网络错误，请重试")
    } finally {
      if (needCaptcha && !success) resetCaptcha()
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4 py-10">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <h1 className="text-2xl font-bold">Zboard</h1>
          <p className="text-muted-foreground mt-1">重置账户密码</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="email" className="text-sm font-medium">注册邮箱</label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
              disabled={done}
              className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-60"
              placeholder="you@example.com"
            />
          </div>

          {needCaptcha && (
            <Captcha
              key={captchaKey}
              provider={settings.captcha_provider as any}
              siteKey={settings.captcha_site_key || ""}
              mode={(settings.turnstile_mode as any) || "managed"}
              onToken={handleToken}
              onError={(msg) => setError(msg)}
            />
          )}

          <div>
            <label htmlFor="code" className="text-sm font-medium">邮箱验证码</label>
            <div className="mt-1 flex gap-2">
              <input
                id="code"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                required
                inputMode="numeric"
                maxLength={6}
                autoComplete="one-time-code"
                disabled={done}
                className="flex-1 min-w-0 rounded-lg border border-border bg-background px-3 py-2 text-sm tracking-widest focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-60"
                placeholder="6 位验证码"
              />
              <Button
                type="button"
                variant="outline"
                onClick={handleSendCode}
                disabled={sending || cooldown > 0 || done}
                className="shrink-0 min-w-[110px]"
              >
                {cooldown > 0 ? `${cooldown}s` : sending ? "发送中" : "获取验证码"}
              </Button>
            </div>
          </div>

          <div>
            <label htmlFor="new-password" className="text-sm font-medium">新密码</label>
            <input
              id="new-password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              minLength={6}
              autoComplete="new-password"
              disabled={done}
              className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-60"
              placeholder="至少 6 位"
            />
          </div>

          <div>
            <label htmlFor="confirm-new" className="text-sm font-medium">确认新密码</label>
            <input
              id="confirm-new"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              minLength={6}
              autoComplete="new-password"
              disabled={done}
              className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-60"
              placeholder="再输入一次"
            />
          </div>

          {error && <p className="text-sm text-red-500" role="alert">{error}</p>}
          {hint && !error && <p className="text-sm text-emerald-600">{hint}</p>}

          <Button type="submit" className="w-full" disabled={loading || done}>
            {loading ? "重置中..." : done ? "已重置" : "重置密码"}
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          想起密码了？
          <Link href="/login" className="ml-1 text-primary hover:underline">
            返回登录
          </Link>
        </p>
      </div>
    </div>
  )
}
