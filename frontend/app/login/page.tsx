"use client"

import { useCallback, useEffect, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { ArrowRight, Lock, Mail, Zap } from "lucide-react"
import { Captcha, captchaEnabled } from "@/components/captcha"
import LanguageSwitcher from "@/components/LanguageSwitcher"
import { Button } from "@/components/ui/button"
import { ApiError, getPublicSettings, login } from "@/lib/api"
import { authCopy } from "@/lib/i18n/auth"
import { useI18n } from "@/lib/i18n/context"

export default function LoginPage() {
  const router = useRouter()
  const { locale } = useI18n()
  const copy = authCopy(locale)
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [settings, setSettings] = useState<Record<string, string>>({})
  const [captchaToken, setCaptchaToken] = useState("")
  const [captchaKey, setCaptchaKey] = useState(0)
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    getPublicSettings()
      .then((res) => setSettings(res.settings || {}))
      .catch(() => {})
  }, [])

  const needCaptcha = captchaEnabled(settings, "login")
  const handleToken = useCallback((token: string) => setCaptchaToken(token), [])
  const resetCaptcha = useCallback(() => {
    setCaptchaToken("")
    setCaptchaKey((value) => value + 1)
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    if (needCaptcha && !captchaToken) {
      setError(copy.common.captchaRequired)
      return
    }
    setLoading(true)
    try {
      await login(email, password, captchaToken)
      router.push("/dashboard")
    } catch (err) {
      setError(err instanceof ApiError ? err.message : copy.common.networkError)
    } finally {
      if (needCaptcha) resetCaptcha()
      setLoading(false)
    }
  }

  return (
    <div className="relative min-h-dvh flex items-center justify-center bg-background px-4 overflow-hidden select-none">
      <div className="absolute right-4 top-4 z-20 sm:right-6 sm:top-6">
        <LanguageSwitcher align="right" side="bottom" />
      </div>
      <div className="absolute top-[-20%] left-[-20%] w-[60%] h-[60%] rounded-full bg-primary/10 blur-[120px] pointer-events-none" />
      <div className="absolute bottom-[-20%] right-[-20%] w-[60%] h-[60%] rounded-full bg-chart-3/10 blur-[140px] pointer-events-none" />
      <div className="absolute top-[45%] right-[-10%] w-[35%] h-[35%] rounded-full bg-chart-2/5 blur-[100px] pointer-events-none" />
      <div className="absolute inset-0 bg-[linear-gradient(to_right,var(--border)_1.5px,transparent_1.5px),linear-gradient(to_bottom,var(--border)_1.5px,transparent_1.5px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_50%,#000_70%,transparent_100%)] opacity-40 pointer-events-none" />

      <div className="relative w-full max-w-md z-10 animate-in fade-in-50 slide-in-from-bottom-6 duration-700">
        <div className="text-center mb-8 flex flex-col items-center">
          <div className="w-12 h-12 rounded-2xl btn-gradient flex items-center justify-center shadow-md shadow-primary/10 mb-3.5 hover:scale-105 transition-transform duration-300">
            <Zap className="w-6 h-6 text-primary-foreground" strokeWidth={2.5} />
          </div>
          <h1 className="font-display text-2xl font-bold tracking-tight text-foreground">{copy.common.brand}</h1>
          <p className="text-sm text-muted-foreground mt-1.5 font-medium">{copy.login.subtitle}</p>
        </div>

        <div className="rounded-2xl border border-border bg-card/70 backdrop-blur-xl p-7 sm:p-8 card-shadow-hover">
          <form onSubmit={handleSubmit} className="space-y-5">
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
                  className="block w-full rounded-xl border border-input bg-background/80 pl-10.5 pr-4 py-3 text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-ring transition-all"
                  placeholder="you@example.com"
                />
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label htmlFor="password" className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {copy.common.password}
                </label>
                <Link href="/forgot-password" className="text-xs text-primary hover:text-primary/80 font-semibold transition-colors">
                  {copy.login.forgot}
                </Link>
              </div>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none z-10">
                  <Lock className="h-4.5 w-4.5 text-muted-foreground group-focus-within:text-primary transition-colors" />
                </div>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  minLength={6}
                  autoComplete="current-password"
                  className="block w-full rounded-xl border border-input bg-background/80 pl-10.5 pr-4 py-3 text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-ring transition-all"
                  placeholder="••••••••"
                />
              </div>
            </div>

            {needCaptcha && (
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

            {error && (
              <div className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-xs text-destructive font-medium animate-in fade-in-0 duration-300" role="alert">
                {error}
              </div>
            )}

            <Button
              type="submit"
              className="w-full rounded-xl btn-gradient hover:opacity-95 text-primary-foreground font-semibold py-3.5 shadow-md shadow-primary/10 flex items-center justify-center gap-1.5"
              disabled={loading}
            >
              <span>{loading ? copy.login.loading : copy.login.submit}</span>
              {!loading && <ArrowRight className="w-4 h-4" />}
            </Button>
          </form>

          <div className="text-center mt-6 text-sm text-muted-foreground border-t border-border pt-5 font-medium">
            {copy.login.registerPrompt}
            <Link href="/register" className="ml-1.5 text-primary hover:text-primary/80 font-semibold transition-colors">
              {copy.login.registerLink}
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
