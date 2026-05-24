"use client"

import { useCallback, useEffect, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { login, getPublicSettings, ApiError } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Captcha, captchaEnabled } from "@/components/captcha"
import { Mail, Lock, Zap, ArrowRight } from "lucide-react"

export default function LoginPage() {
  const router = useRouter()
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
  const handleToken = useCallback((t: string) => setCaptchaToken(t), [])
  const resetCaptcha = useCallback(() => {
    setCaptchaToken("")
    setCaptchaKey((v) => v + 1)
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    if (needCaptcha && !captchaToken) {
      setError("请先完成安全验证")
      return
    }
    setLoading(true)
    try {
      await login(email, password, captchaToken)
      router.push("/dashboard")
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "网络错误，请稍后重试")
    } finally {
      if (needCaptcha) resetCaptcha()
      setLoading(false)
    }
  }

  return (
    <div className="relative min-h-screen flex items-center justify-center bg-slate-950 px-4 overflow-hidden select-none">
      {/* 极美科技流光背景粒子 */}
      <div className="absolute top-[-20%] left-[-20%] w-[60%] h-[60%] rounded-full bg-blue-600/10 blur-[120px] pointer-events-none" />
      <div className="absolute bottom-[-20%] right-[-20%] w-[60%] h-[60%] rounded-full bg-indigo-600/15 blur-[140px] pointer-events-none" />
      <div className="absolute top-[40%] right-[-10%] w-[35%] h-[35%] rounded-full bg-sky-500/5 blur-[100px] pointer-events-none" />
      
      {/* 网格背景线，增添科技感 */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#0f172a_1px,transparent_1px),linear-gradient(to_bottom,#0f172a_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_50%,#000_70%,transparent_100%)] opacity-40 pointer-events-none" />

      <div className="relative w-full max-w-md z-10 animate-in fade-in-50 slide-in-from-bottom-6 duration-700">
        
        {/* Logo 区域 */}
        <div className="text-center mb-8 flex flex-col items-center">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-tr from-blue-600 via-indigo-600 to-sky-400 flex items-center justify-center shadow-lg shadow-blue-500/20 mb-3.5 hover:scale-105 transition-transform duration-300">
            <Zap className="w-6 h-6 text-white" strokeWidth={2.5} />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-white">Zboard</h1>
          <p className="text-sm text-slate-400 mt-1.5">进入多端协同云加速控制中心</p>
        </div>

        {/* 悬浮毛玻璃表单卡片 */}
        <div className="rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xl p-7 sm:p-8 shadow-[0_20px_50px_rgba(0,0,0,0.3)]">
          <form onSubmit={handleSubmit} className="space-y-5">
            {/* 邮箱输入框 */}
            <div>
              <label htmlFor="email" className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1.5">
                接入邮箱
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
                  className="block w-full rounded-xl border border-white/10 bg-slate-950/60 pl-10.5 pr-4 py-3 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/40 focus:border-blue-500/60 transition-all"
                  placeholder="you@example.com"
                />
              </div>
            </div>

            {/* 密码输入框 */}
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label htmlFor="password" className="block text-xs font-semibold uppercase tracking-wider text-slate-400">
                  访问密码
                </label>
                <Link href="/forgot-password" className="text-xs text-blue-400 hover:text-blue-300 font-medium transition-colors">
                  忘记密码？
                </Link>
              </div>
              <div className="relative rounded-xl overflow-hidden group">
                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                  <Lock className="h-4.5 w-4.5 text-slate-500 group-focus-within:text-blue-500 transition-colors" />
                </div>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  minLength={6}
                  autoComplete="current-password"
                  className="block w-full rounded-xl border border-white/10 bg-slate-950/60 pl-10.5 pr-4 py-3 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/40 focus:border-blue-500/60 transition-all"
                  placeholder="••••••••"
                />
              </div>
            </div>

            {/* 安全验证 */}
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

            {/* 错误提示 */}
            {error && (
              <div className="rounded-xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-xs text-red-400 animate-in fade-in-0 duration-300" role="alert">
                {error}
              </div>
            )}

            {/* 炫彩流光登录按钮 */}
            <Button
              type="submit"
              className="w-full rounded-xl bg-gradient-to-r from-blue-600 via-indigo-600 to-sky-500 hover:opacity-95 text-white font-medium py-3.5 shadow-lg shadow-blue-500/20 flex items-center justify-center gap-1.5 active:scale-[0.99] transition-all"
              disabled={loading}
            >
              <span>{loading ? "正在接入控制中心..." : "验证接入"}</span>
              {!loading && <ArrowRight className="w-4 h-4" />}
            </Button>
          </form>

          {/* 底端注册导航 */}
          <div className="text-center mt-6 text-sm text-slate-400 border-t border-white/5 pt-5">
            还没有接入账号？
            <Link href="/register" className="ml-1.5 text-blue-400 hover:text-blue-300 font-semibold transition-colors">
              立即加入
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
