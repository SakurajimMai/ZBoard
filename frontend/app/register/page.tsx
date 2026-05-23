"use client"

import { useCallback, useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { login, registerWithCode, sendEmailCode, getPublicSettings, ApiError } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Captcha, captchaEnabled } from "@/components/captcha"

const COOLDOWN_SECONDS = 120

export default function RegisterPage() {
  const router = useRouter()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [code, setCode] = useState("")
  const [settings, setSettings] = useState<Record<string, string>>({})
  const [allowRegister, setAllowRegister] = useState(true)
  const [captchaToken, setCaptchaToken] = useState("")
  const [captchaKey, setCaptchaKey] = useState(0)
  const [cooldown, setCooldown] = useState(0)
  const [sending, setSending] = useState(false)
  const [error, setError] = useState("")
  const [hint, setHint] = useState("")
  const [loading, setLoading] = useState(false)
  const timerRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    getPublicSettings()
      .then((res) => {
        const s = res.settings || {}
        setSettings(s)
        setAllowRegister(!(s.allow_register === "0" || s.allow_register === "false"))
      })
      .catch(() => {})
    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
    }
  }, [])

  const needCaptcha = captchaEnabled(settings, "register")
  const loginNeedsCaptcha = captchaEnabled(settings, "login")
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
      await sendEmailCode(email.trim(), "register", captchaToken)
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
    if (!allowRegister) {
      setError("当前站点已关闭用户注册")
      return
    }
    if (password !== confirmPassword) {
      setError("两次输入的密码不一致")
      return
    }
    if (password.length < 6) {
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
    try {
      await registerWithCode(email.trim(), password, code.trim(), captchaToken)
      if (loginNeedsCaptcha) {
        router.push("/login")
        return
      }
      await login(email.trim(), password)
      router.push("/dashboard")
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "网络错误，请重试")
    } finally {
      if (needCaptcha) resetCaptcha()
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4 py-10">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <h1 className="text-2xl font-bold">Zboard</h1>
          <p className="text-muted-foreground mt-1">创建新账户</p>
        </div>

        {!allowRegister && (
          <div className="rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-800">
            当前站点已关闭用户注册。
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="email" className="text-sm font-medium">邮箱</label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
              className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label htmlFor="password" className="text-sm font-medium">密码</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={6}
              autoComplete="new-password"
              className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="至少 6 位"
            />
          </div>

          <div>
            <label htmlFor="confirm" className="text-sm font-medium">确认密码</label>
            <input
              id="confirm"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              minLength={6}
              autoComplete="new-password"
              className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="再输入一次"
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
                className="flex-1 min-w-0 rounded-lg border border-border bg-background px-3 py-2 text-sm tracking-widest focus:outline-none focus:ring-2 focus:ring-primary"
                placeholder="6 位验证码"
              />
              <Button
                type="button"
                variant="outline"
                onClick={handleSendCode}
                disabled={sending || cooldown > 0 || !allowRegister}
                className="shrink-0 min-w-[110px]"
              >
                {cooldown > 0 ? `${cooldown}s` : sending ? "发送中" : "获取验证码"}
              </Button>
            </div>
          </div>

          {error && <p className="text-sm text-red-500" role="alert">{error}</p>}
          {hint && !error && <p className="text-sm text-emerald-600">{hint}</p>}

          <Button type="submit" className="w-full" disabled={loading || !allowRegister}>
            {loading ? "注册中..." : "注册"}
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          已有账户？
          <Link href="/login" className="ml-1 text-primary hover:underline">
            返回登录
          </Link>
        </p>
      </div>
    </div>
  )
}
