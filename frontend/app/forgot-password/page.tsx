"use client"

import { useCallback, useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { resetPassword, sendEmailCode, getPublicSettings, ApiError } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Captcha, captchaEnabled } from "@/components/captcha"
import { Mail, Lock, Zap, Shield, ArrowRight } from "lucide-react"

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
      setError("请先填写邮箱账号")
      return
    }
    if (needCaptcha && !captchaToken) {
      setError("请先完成安全验证")
      return
    }
    setSending(true)
    try {
      await sendEmailCode(email.trim(), "reset_password", captchaToken)
      setHint("重置密码验证码已发送，请注意查收邮件")
      startCooldown()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "凭证发送失败，请重试")
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
      setError("密码安全强度不够，至少 6 位")
      return
    }
    if (!code.trim()) {
      setError("请填写邮箱验证码")
      return
    }
    if (needCaptcha && !captchaToken) {
      setError("请先完成安全验证")
      return
    }
    setLoading(true)
    let success = false
    try {
      await resetPassword(email.trim(), newPassword, code.trim(), captchaToken)
      success = true
      setDone(true)
      setHint("重置成功！控制端凭证已生效，正在为您跳转至登录页...")
      setTimeout(() => router.push("/login"), 2500)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "重置请求失败，请重试")
    } finally {
      if (needCaptcha && !success) resetCaptcha()
      setLoading(false)
    }
  }

  return (
    <div className="relative min-h-screen flex items-center justify-center bg-slate-950 px-4 py-12 overflow-hidden select-none">
      {/* 极美科技流光背景粒子 */}
      <div className="absolute top-[-20%] left-[-20%] w-[60%] h-[60%] rounded-full bg-blue-600/10 blur-[120px] pointer-events-none" />
      <div className="absolute bottom-[-20%] right-[-20%] w-[60%] h-[60%] rounded-full bg-indigo-600/15 blur-[140px] pointer-events-none" />
      <div className="absolute top-[40%] right-[-10%] w-[35%] h-[35%] rounded-full bg-sky-500/5 blur-[100px] pointer-events-none" />
      
      {/* 网格背景线 */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#0f172a_1px,transparent_1px),linear-gradient(to_bottom,#0f172a_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_50%,#000_70%,transparent_100%)] opacity-40 pointer-events-none" />

      <div className="relative w-full max-w-md z-10 animate-in fade-in-50 slide-in-from-bottom-6 duration-700">
        
        {/* Logo 区域 */}
        <div className="text-center mb-8 flex flex-col items-center">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-tr from-blue-600 via-indigo-600 to-sky-400 flex items-center justify-center shadow-lg shadow-blue-500/20 mb-3.5 hover:scale-105 transition-transform duration-300">
            <Zap className="w-6 h-6 text-white" strokeWidth={2.5} />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-white">Zboard</h1>
          <p className="text-sm text-slate-400 mt-1.5">重置账户密码 · 快速恢复控制权</p>
        </div>

        {/* 悬浮毛玻璃表单卡片 */}
        <div className="rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xl p-7 sm:p-8 shadow-[0_20px_50px_rgba(0,0,0,0.3)]">
          <form onSubmit={handleSubmit} className="space-y-4">
            
            {/* 邮箱输入框 */}
            <div>
              <label htmlFor="email" className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5">
                安全邮箱
              </label>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                  <Mail className="h-4.5 w-4.5 text-slate-500 group-focus-within:text-blue-500 transition-colors" />
                </div>
                <input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoComplete="email"
                  disabled={done}
                  className="block w-full rounded-xl border border-white/10 bg-slate-950/60 pl-10.5 pr-4 py-3 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/40 focus:border-blue-500/60 transition-all disabled:opacity-50"
                  placeholder="you@example.com"
                />
              </div>
            </div>

            {/* 人机安全验证 */}
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

            {/* 验证码验证 */}
            <div>
              <label htmlFor="code" className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5">
                接收验证凭证
              </label>
              <div className="flex gap-2.5">
                <div className="relative flex-1 rounded-xl overflow-hidden group">
                  <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                    <Shield className="h-4.5 w-4.5 text-slate-500 group-focus-within:text-blue-500 transition-colors" />
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
                    className="block w-full rounded-xl border border-white/10 bg-slate-950/60 pl-10.5 pr-4 py-3 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/40 focus:border-blue-500/60 tracking-widest text-center transition-all disabled:opacity-50"
                    placeholder="6 位凭证"
                  />
                </div>
                
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleSendCode}
                  disabled={sending || cooldown > 0 || done}
                  className="rounded-xl border border-white/10 bg-slate-950/40 hover:bg-slate-950/60 text-slate-300 font-semibold px-4.5 py-3.5 text-xs tracking-wider shrink-0 transition-all active:scale-[0.98] min-w-[120px] disabled:opacity-50"
                >
                  {cooldown > 0 ? `${cooldown}s 重新发送` : sending ? "投递中..." : "获取凭证"}
                </Button>
              </div>
            </div>

            {/* 新密码 */}
            <div>
              <label htmlFor="new-password" className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5">
                设置新密码
              </label>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                  <Lock className="h-4.5 w-4.5 text-slate-500 group-focus-within:text-blue-500 transition-colors" />
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
                  className="block w-full rounded-xl border border-white/10 bg-slate-950/60 pl-10.5 pr-4 py-3 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/40 focus:border-blue-500/60 transition-all disabled:opacity-50"
                  placeholder="至少 6 位新密码"
                />
              </div>
            </div>

            {/* 确认新密码 */}
            <div>
              <label htmlFor="confirm-new" className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5">
                确认新密码
              </label>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                  <Lock className="h-4.5 w-4.5 text-slate-500 group-focus-within:text-blue-500 transition-colors" />
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
                  className="block w-full rounded-xl border border-white/10 bg-slate-950/60 pl-10.5 pr-4 py-3 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/40 focus:border-blue-500/60 transition-all disabled:opacity-50"
                  placeholder="再次输入以核对"
                />
              </div>
            </div>

            {/* 错误 & 提示信息 */}
            {error && (
              <div className="rounded-xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-xs text-red-400 animate-in fade-in-0 duration-300" role="alert">
                {error}
              </div>
            )}
            {hint && (
              <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/10 px-4 py-3 text-xs text-emerald-400 animate-in fade-in-0 duration-300">
                {hint}
              </div>
            )}

            {/* 提交按钮 */}
            <Button
              type="submit"
              className="w-full rounded-xl bg-gradient-to-r from-blue-600 via-indigo-600 to-sky-500 hover:opacity-95 text-white font-medium py-3.5 shadow-lg shadow-blue-500/20 flex items-center justify-center gap-1.5 active:scale-[0.99] transition-all disabled:opacity-50"
              disabled={loading || done}
            >
              <span>{loading ? "正在应用凭证..." : done ? "重置生效" : "重置访问密码"}</span>
              {!loading && !done && <ArrowRight className="w-4 h-4" />}
            </Button>
          </form>

          {/* 底部导航 */}
          <div className="text-center mt-6 text-sm text-slate-400 border-t border-white/5 pt-5">
            想起密码了？
            <Link href="/login" className="ml-1.5 text-blue-400 hover:text-blue-300 font-semibold transition-colors">
              返回登录
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
