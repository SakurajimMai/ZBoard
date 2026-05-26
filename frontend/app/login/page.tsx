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
    <div className="relative min-h-screen flex items-center justify-center bg-[#f8fafc] px-4 overflow-hidden select-none">
      <div className="absolute right-4 top-4 z-20 sm:right-6 sm:top-6">
        <LanguageSwitcher align="right" side="bottom" />
      </div>
      <div className="absolute top-[-20%] left-[-20%] w-[60%] h-[60%] rounded-full bg-blue-400/10 blur-[120px] pointer-events-none" />
      <div className="absolute bottom-[-20%] right-[-20%] w-[60%] h-[60%] rounded-full bg-indigo-400/12 blur-[140px] pointer-events-none" />
      <div className="absolute top-[45%] right-[-10%] w-[35%] h-[35%] rounded-full bg-pink-400/5 blur-[100px] pointer-events-none" />
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#f1f5f9_1.5px,transparent_1.5px),linear-gradient(to_bottom,#f1f5f9_1.5px,transparent_1.5px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_50%,#000_70%,transparent_100%)] opacity-70 pointer-events-none" />

      <div className="relative w-full max-w-md z-10 animate-in fade-in-50 slide-in-from-bottom-6 duration-700">
        <div className="text-center mb-8 flex flex-col items-center">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-tr from-blue-600 via-indigo-600 to-sky-400 flex items-center justify-center shadow-md shadow-blue-500/10 mb-3.5 hover:scale-105 transition-transform duration-300">
            <Zap className="w-6 h-6 text-white" strokeWidth={2.5} />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900">{copy.common.brand}</h1>
          <p className="text-sm text-slate-500 mt-1.5 font-medium">{copy.login.subtitle}</p>
        </div>

        <div className="rounded-2xl border border-white bg-white/70 backdrop-blur-xl p-7 sm:p-8 shadow-[0_20px_50px_rgba(15,23,42,0.06)]">
          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label htmlFor="email" className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5">
                {copy.common.email}
              </label>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                  <Mail className="h-4.5 w-4.5 text-slate-400 group-focus-within:text-blue-500 transition-colors" />
                </div>
                <input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoComplete="email"
                  className="block w-full rounded-xl border border-slate-200/80 bg-white/80 pl-10.5 pr-4 py-3 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
                  placeholder="you@example.com"
                />
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label htmlFor="password" className="block text-xs font-semibold uppercase tracking-wider text-slate-400">
                  {copy.common.password}
                </label>
                <Link href="/forgot-password" className="text-xs text-blue-600 hover:text-blue-500 font-semibold transition-colors">
                  {copy.login.forgot}
                </Link>
              </div>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                  <Lock className="h-4.5 w-4.5 text-slate-400 group-focus-within:text-blue-500 transition-colors" />
                </div>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  minLength={6}
                  autoComplete="current-password"
                  className="block w-full rounded-xl border border-slate-200/80 bg-white/80 pl-10.5 pr-4 py-3 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
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
              <div className="rounded-xl border border-red-500/10 bg-red-500/5 px-4 py-3 text-xs text-red-500 font-medium animate-in fade-in-0 duration-300" role="alert">
                {error}
              </div>
            )}

            <Button
              type="submit"
              className="w-full rounded-xl bg-gradient-to-r from-blue-600 via-indigo-600 to-sky-500 hover:opacity-95 text-white font-semibold py-3.5 shadow-md shadow-blue-500/10 flex items-center justify-center gap-1.5 active:scale-[0.99] transition-all"
              disabled={loading}
            >
              <span>{loading ? copy.login.loading : copy.login.submit}</span>
              {!loading && <ArrowRight className="w-4 h-4" />}
            </Button>
          </form>

          <div className="text-center mt-6 text-sm text-slate-500 border-t border-slate-100 pt-5 font-medium">
            {copy.login.registerPrompt}
            <Link href="/register" className="ml-1.5 text-blue-600 hover:text-blue-500 font-semibold transition-colors">
              {copy.login.registerLink}
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
