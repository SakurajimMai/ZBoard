"use client"

import { useCallback, useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { ArrowRight, Lock, Mail, Shield, Zap } from "lucide-react"
import { Captcha, captchaEnabled } from "@/components/captcha"
import LanguageSwitcher from "@/components/LanguageSwitcher"
import { Button } from "@/components/ui/button"
import { ApiError, getPublicSettings, resetPassword, sendEmailCode } from "@/lib/api"
import { authCopy } from "@/lib/i18n/auth"
import { useI18n } from "@/lib/i18n/context"

const COOLDOWN_SECONDS = 120

export default function ForgotPasswordPage() {
  const router = useRouter()
  const { locale } = useI18n()
  const copy = authCopy(locale)
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
  const handleToken = useCallback((token: string) => setCaptchaToken(token), [])
  const resetCaptcha = useCallback(() => {
    setCaptchaToken("")
    setCaptchaKey((value) => value + 1)
  }, [])

  const startCooldown = () => {
    setCooldown(COOLDOWN_SECONDS)
    if (timerRef.current) clearInterval(timerRef.current)
    timerRef.current = setInterval(() => {
      setCooldown((previous) => {
        if (previous <= 1) {
          if (timerRef.current) clearInterval(timerRef.current)
          return 0
        }
        return previous - 1
      })
    }, 1000)
  }

  const handleSendCode = async () => {
    setError("")
    setHint("")
    if (!email.trim()) {
      setError(copy.register.emailRequired)
      return
    }
    if (needCaptcha && !captchaToken) {
      setError(copy.common.captchaRequired)
      return
    }
    setSending(true)
    try {
      await sendEmailCode(email.trim(), "reset_password", captchaToken)
      setHint(copy.forgot.sendSuccess)
      startCooldown()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : copy.forgot.sendError)
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
      setError(copy.forgot.passwordMismatch)
      return
    }
    if (newPassword.length < 6) {
      setError(copy.forgot.passwordWeak)
      return
    }
    if (!code.trim()) {
      setError(copy.forgot.codeRequired)
      return
    }
    if (needCaptcha && !captchaToken) {
      setError(copy.common.captchaRequired)
      return
    }
    setLoading(true)
    let success = false
    try {
      await resetPassword(email.trim(), newPassword, code.trim(), captchaToken)
      success = true
      setDone(true)
      setHint(copy.forgot.success)
      setTimeout(() => router.push("/login"), 2500)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : copy.forgot.submitError)
    } finally {
      if (needCaptcha && !success) resetCaptcha()
      setLoading(false)
    }
  }

  return (
    <div className="relative min-h-dvh flex items-center justify-center bg-background px-4 py-12 overflow-hidden select-none">
      <div className="absolute right-4 top-4 z-20 sm:right-6 sm:top-6">
        <LanguageSwitcher align="right" side="bottom" />
      </div>
      <div className="absolute top-[-20%] left-[-20%] w-[60%] h-[60%] rounded-full bg-primary/10 blur-[120px] pointer-events-none" />
      <div className="absolute bottom-[-20%] right-[-20%] w-[60%] h-[60%] rounded-full bg-chart-3/10 blur-[140px] pointer-events-none" />
      <div className="absolute top-[40%] right-[-10%] w-[35%] h-[35%] rounded-full bg-chart-2/5 blur-[100px] pointer-events-none" />
      <div className="absolute inset-0 bg-[linear-gradient(to_right,var(--border)_1.5px,transparent_1.5px),linear-gradient(to_bottom,var(--border)_1.5px,transparent_1.5px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_50%,#000_70%,transparent_100%)] opacity-40 pointer-events-none" />

      <div className="relative w-full max-w-md z-10 animate-in fade-in-50 slide-in-from-bottom-6 duration-700">
        <div className="text-center mb-8 flex flex-col items-center">
          <div className="w-12 h-12 rounded-2xl btn-gradient flex items-center justify-center shadow-md shadow-primary/10 mb-3.5 hover:scale-105 transition-transform duration-300">
            <Zap className="w-6 h-6 text-primary-foreground" strokeWidth={2.5} />
          </div>
          <h1 className="font-display text-2xl font-bold tracking-tight text-foreground">{copy.common.brand}</h1>
          <p className="text-sm text-muted-foreground mt-1.5 font-medium">{copy.forgot.subtitle}</p>
        </div>

        <div className="rounded-2xl border border-border bg-card/70 backdrop-blur-xl p-7 sm:p-8 card-shadow-hover">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="email" className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                {copy.common.email}
              </label>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none z-10">
                  <Mail className="h-4.5 w-4.5 text-muted-foreground group-focus-within:text-primary transition-colors" />
                </div>
                <input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoComplete="email"
                  disabled={done}
                  className="block w-full rounded-xl border border-input bg-background/80 pl-10.5 pr-4 py-3 text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-ring transition-all disabled:opacity-50"
                  placeholder="you@example.com"
                />
              </div>
            </div>

            {needCaptcha && !done && (
              <div className="py-1">
                <Captcha
                  key={captchaKey}
                  provider={settings.captcha_provider as any}
                  siteKey={settings.captcha_site_key || ""}
                  mode={(settings.turnstile_mode as any) || "managed"}
                  onToken={handleToken}
                  onError={(msg) => setError(msg)}
                />
              </div>
            )}

            <div>
              <label htmlFor="code" className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                {copy.forgot.codeLabel}
              </label>
              <div className="flex gap-2.5">
                <div className="relative flex-1 rounded-xl overflow-hidden group">
                  <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none z-10">
                    <Shield className="h-4.5 w-4.5 text-muted-foreground group-focus-within:text-primary transition-colors" />
                  </div>
                  <input
                    id="code"
                    value={code}
                    onChange={(e) => setCode(e.target.value)}
                    required
                    inputMode="numeric"
                    maxLength={6}
                    autoComplete="one-time-code"
                    disabled={done}
                    className="block w-full rounded-xl border border-input bg-background/80 pl-10.5 pr-4 py-3 text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-ring tracking-widest text-center transition-all disabled:opacity-50"
                    placeholder={copy.forgot.codePlaceholder}
                  />
                </div>

                <Button
                  type="button"
                  variant="outline"
                  onClick={handleSendCode}
                  disabled={sending || cooldown > 0 || done}
                  className="rounded-xl border border-input bg-background/80 hover:bg-accent text-foreground font-semibold px-4.5 py-3.5 text-xs tracking-wider shrink-0 transition-all min-w-[120px] disabled:opacity-50"
                >
                  {cooldown > 0 ? `${cooldown}s ${copy.forgot.resend}` : sending ? copy.forgot.sending : copy.forgot.sendCode}
                </Button>
              </div>
            </div>

            <div>
              <label htmlFor="new-password" className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                {copy.common.newPassword}
              </label>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none z-10">
                  <Lock className="h-4.5 w-4.5 text-muted-foreground group-focus-within:text-primary transition-colors" />
                </div>
                <input
                  id="new-password"
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  required
                  minLength={6}
                  autoComplete="new-password"
                  disabled={done}
                  className="block w-full rounded-xl border border-input bg-background/80 pl-10.5 pr-4 py-3 text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-ring transition-all disabled:opacity-50"
                  placeholder={copy.register.passwordPlaceholder}
                />
              </div>
            </div>

            <div>
              <label htmlFor="confirm-new" className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                {copy.common.confirmPassword}
              </label>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none z-10">
                  <Lock className="h-4.5 w-4.5 text-muted-foreground group-focus-within:text-primary transition-colors" />
                </div>
                <input
                  id="confirm-new"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  minLength={6}
                  autoComplete="new-password"
                  disabled={done}
                  className="block w-full rounded-xl border border-input bg-background/80 pl-10.5 pr-4 py-3 text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-ring transition-all disabled:opacity-50"
                  placeholder={copy.register.confirmPlaceholder}
                />
              </div>
            </div>

            {error && (
              <div className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-xs text-destructive font-medium animate-in fade-in-0 duration-300" role="alert">
                {error}
              </div>
            )}
            {hint && (
              <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 px-4 py-3 text-xs text-emerald-600 dark:text-emerald-400 font-medium animate-in fade-in-0 duration-300">
                {hint}
              </div>
            )}

            <Button
              type="submit"
              className="w-full rounded-xl btn-gradient hover:opacity-95 text-primary-foreground font-semibold py-3.5 shadow-md shadow-primary/10 flex items-center justify-center gap-1.5 disabled:opacity-50"
              disabled={loading || done}
            >
              <span>{loading ? copy.forgot.loading : done ? copy.forgot.done : copy.forgot.submit}</span>
              {!loading && !done && <ArrowRight className="w-4 h-4" />}
            </Button>
          </form>

          <div className="text-center mt-6 text-sm text-muted-foreground border-t border-border pt-5 font-medium">
            {copy.forgot.backPrompt}
            <Link href="/login" className="ml-1.5 text-primary hover:text-primary/80 font-semibold transition-colors">
              {copy.forgot.backLink}
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
