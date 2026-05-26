"use client"

import { useCallback, useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { ArrowRight, Lock, Mail, Shield, Zap } from "lucide-react"
import { Captcha, captchaEnabled } from "@/components/captcha"
import LanguageSwitcher from "@/components/LanguageSwitcher"
import { Button } from "@/components/ui/button"
import { ApiError, getPublicSettings, login, register, registerWithCode, sendEmailCode } from "@/lib/api"
import { authCopy } from "@/lib/i18n/auth"
import { useI18n } from "@/lib/i18n/context"

const COOLDOWN_SECONDS = 120

function truthy(value?: string) {
  return value === "1" || value === "true"
}

export default function RegisterPage() {
  const router = useRouter()
  const { locale } = useI18n()
  const copy = authCopy(locale)
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
        const publicSettings = res.settings || {}
        setSettings(publicSettings)
        setAllowRegister(!(publicSettings.allow_register === "0" || publicSettings.allow_register === "false"))
      })
      .catch(() => {})
    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
    }
  }, [])

  const requireEmailVerify = truthy(settings.require_email_verify) && truthy(settings.email_verify_available)
  const needCaptcha = captchaEnabled(settings, "register")
  const loginNeedsCaptcha = captchaEnabled(settings, "login")
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
    if (!requireEmailVerify) return
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
      await sendEmailCode(email.trim(), "register", captchaToken)
      setHint(copy.register.sendSuccess)
      startCooldown()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : copy.register.sendError)
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
      setError(copy.register.disabled)
      return
    }
    if (password !== confirmPassword) {
      setError(copy.register.passwordMismatch)
      return
    }
    if (password.length < 6) {
      setError(copy.register.passwordWeak)
      return
    }
    if (requireEmailVerify && !code.trim()) {
      setError(copy.register.codeRequired)
      return
    }
    if (needCaptcha && !captchaToken) {
      setError(copy.common.captchaRequired)
      return
    }
    setLoading(true)
    try {
      if (requireEmailVerify) {
        await registerWithCode(email.trim(), password, code.trim(), captchaToken)
      } else {
        await register(email.trim(), password, captchaToken)
      }
      if (loginNeedsCaptcha) {
        router.push("/login")
        return
      }
      await login(email.trim(), password)
      router.push("/dashboard")
    } catch (err) {
      setError(err instanceof ApiError ? err.message : copy.register.submitError)
    } finally {
      if (needCaptcha) resetCaptcha()
      setLoading(false)
    }
  }

  return (
    <div className="relative min-h-screen flex items-center justify-center bg-[#f8fafc] px-4 py-12 overflow-hidden select-none">
      <div className="absolute right-4 top-4 z-20 sm:right-6 sm:top-6">
        <LanguageSwitcher align="right" side="bottom" />
      </div>
      <div className="absolute top-[-20%] right-[-20%] w-[60%] h-[60%] rounded-full bg-blue-400/10 blur-[120px] pointer-events-none" />
      <div className="absolute bottom-[-20%] left-[-20%] w-[60%] h-[60%] rounded-full bg-indigo-400/12 blur-[140px] pointer-events-none" />
      <div className="absolute top-[30%] left-[-10%] w-[35%] h-[35%] rounded-full bg-pink-400/5 blur-[100px] pointer-events-none" />
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#f1f5f9_1.5px,transparent_1.5px),linear-gradient(to_bottom,#f1f5f9_1.5px,transparent_1.5px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_50%,#000_70%,transparent_100%)] opacity-70 pointer-events-none" />

      <div className="relative w-full max-w-md z-10 animate-in fade-in-50 slide-in-from-bottom-6 duration-700">
        <div className="text-center mb-8 flex flex-col items-center">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-tr from-blue-600 via-indigo-600 to-sky-400 flex items-center justify-center shadow-md shadow-blue-500/10 mb-3.5 hover:scale-105 transition-transform duration-300">
            <Zap className="w-6 h-6 text-white" strokeWidth={2.5} />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900">{copy.common.brand}</h1>
          <p className="text-sm text-slate-500 mt-1.5 font-medium">{copy.register.subtitle}</p>
        </div>

        {!allowRegister && (
          <div className="rounded-xl border border-amber-500/10 bg-amber-500/5 p-4 text-xs text-amber-600 mb-5 animate-in fade-in duration-300 font-medium">
            {copy.register.disabled}
          </div>
        )}

        <div className="rounded-2xl border border-white bg-white/70 backdrop-blur-xl p-7 sm:p-8 shadow-[0_20px_50px_rgba(15,23,42,0.06)]">
          <form onSubmit={handleSubmit} className="space-y-4">
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
                  disabled={!allowRegister}
                  className="block w-full rounded-xl border border-slate-200/80 bg-white/80 pl-10.5 pr-4 py-3 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all disabled:opacity-50"
                  placeholder="you@example.com"
                />
              </div>
            </div>

            <div>
              <label htmlFor="password" className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5">
                {copy.common.password}
              </label>
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
                  autoComplete="new-password"
                  disabled={!allowRegister}
                  className="block w-full rounded-xl border border-slate-200/80 bg-white/80 pl-10.5 pr-4 py-3 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all disabled:opacity-50"
                  placeholder={copy.register.passwordPlaceholder}
                />
              </div>
            </div>

            <div>
              <label htmlFor="confirm" className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5">
                {copy.common.confirmPassword}
              </label>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                  <Lock className="h-4.5 w-4.5 text-slate-400 group-focus-within:text-blue-500 transition-colors" />
                </div>
                <input
                  id="confirm"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  minLength={6}
                  autoComplete="new-password"
                  disabled={!allowRegister}
                  className="block w-full rounded-xl border border-slate-200/80 bg-white/80 pl-10.5 pr-4 py-3 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all disabled:opacity-50"
                  placeholder={copy.register.confirmPlaceholder}
                />
              </div>
            </div>

            {needCaptcha && allowRegister && (
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

            {requireEmailVerify && (
              <div>
                <label htmlFor="code" className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5">
                  {copy.register.codeLabel}
                </label>
                <div className="flex gap-2.5">
                  <div className="relative flex-1 rounded-xl overflow-hidden group">
                    <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                      <Shield className="h-4.5 w-4.5 text-slate-400 group-focus-within:text-blue-500 transition-colors" />
                    </div>
                    <input
                      id="code"
                      value={code}
                      onChange={(e) => setCode(e.target.value)}
                      required={requireEmailVerify}
                      inputMode="numeric"
                      maxLength={6}
                      autoComplete="one-time-code"
                      disabled={!allowRegister}
                      className="block w-full rounded-xl border border-slate-200/80 bg-white/80 pl-10.5 pr-4 py-3 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 tracking-widest text-center transition-all disabled:opacity-50"
                      placeholder={copy.register.codePlaceholder}
                    />
                  </div>

                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleSendCode}
                    disabled={sending || cooldown > 0 || !allowRegister}
                    className="rounded-xl border border-slate-200/80 bg-white/80 hover:bg-slate-100/80 text-slate-600 font-semibold px-4.5 py-3.5 text-xs tracking-wider shrink-0 transition-all active:scale-[0.98] min-w-[120px] disabled:opacity-50"
                  >
                    {cooldown > 0 ? `${cooldown}s ${copy.register.resend}` : sending ? copy.register.sending : copy.register.sendCode}
                  </Button>
                </div>
              </div>
            )}

            {error && (
              <div className="rounded-xl border border-red-500/10 bg-red-500/5 px-4 py-3 text-xs text-red-500 font-medium animate-in fade-in-0 duration-300" role="alert">
                {error}
              </div>
            )}
            {hint && !error && (
              <div className="rounded-xl border border-emerald-500/10 bg-emerald-500/5 px-4 py-3 text-xs text-emerald-600 font-medium animate-in fade-in-0 duration-300">
                {hint}
              </div>
            )}

            <Button
              type="submit"
              className="w-full rounded-xl bg-gradient-to-r from-blue-600 via-indigo-600 to-sky-500 hover:opacity-95 text-white font-semibold py-3.5 shadow-md shadow-blue-500/10 flex items-center justify-center gap-1.5 active:scale-[0.99] transition-all disabled:opacity-50"
              disabled={loading || !allowRegister}
            >
              <span>{loading ? copy.register.loading : copy.register.submit}</span>
              {!loading && <ArrowRight className="w-4 h-4" />}
            </Button>
          </form>

          <div className="text-center mt-6 text-sm text-slate-500 border-t border-slate-100 pt-5 font-medium">
            {copy.register.loginPrompt}
            <Link href="/login" className="ml-1.5 text-blue-600 hover:text-blue-500 font-semibold transition-colors">
              {copy.register.loginLink}
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
