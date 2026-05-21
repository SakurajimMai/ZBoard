"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { login, register, registerWithCode, sendEmailCode, getPublicSettings, ApiError } from "@/lib/api"
import { Button } from "@/components/ui/button"

export default function LoginPage() {
  const router = useRouter()
  const [isLogin, setIsLogin] = useState(true)
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [code, setCode] = useState("")
  const [requireEmailVerify, setRequireEmailVerify] = useState(false)
  const [allowRegister, setAllowRegister] = useState(true)
  const [sendingCode, setSendingCode] = useState(false)
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    getPublicSettings()
      .then((res) => {
        const s = res.settings || {}
        setRequireEmailVerify(s.require_email_verify === "1" || s.require_email_verify === "true")
        setAllowRegister(!(s.allow_register === "0" || s.allow_register === "false"))
      })
      .catch(() => {})
  }, [])

  const handleSendCode = async () => {
    if (!email.trim()) {
      setError("请先填写邮箱")
      return
    }
    setSendingCode(true)
    setError("")
    try {
      await sendEmailCode(email, "register")
      setError("验证码已发送，请检查邮箱")
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "验证码发送失败")
    } finally {
      setSendingCode(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    setLoading(true)
    try {
      if (isLogin) {
        await login(email, password)
        router.push("/dashboard")
      } else {
        if (!allowRegister) {
          setError("当前站点已关闭用户注册")
          return
        }
        if (requireEmailVerify) {
          await registerWithCode(email, password, code)
        } else {
          try {
            await register(email, password)
          } catch (err) {
            if (err instanceof ApiError && err.code === "email_verify_required") {
              setRequireEmailVerify(true)
              setError("请先获取并填写邮箱验证码")
              return
            }
            throw err
          }
        }
        await login(email, password)
        router.push("/dashboard")
      }
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError("网络错误，请重试")
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <h1 className="text-2xl font-bold">Zboard</h1>
          <p className="text-muted-foreground mt-1">
            {isLogin ? "登录您的账户" : "创建新账户"}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="text-sm font-medium">邮箱</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="you@example.com"
            />
          </div>
          <div>
            <label className="text-sm font-medium">密码</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={6}
              className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="••••••••"
            />
          </div>
          {!isLogin && requireEmailVerify && (
            <div>
              <label className="text-sm font-medium">邮箱验证码</label>
              <div className="mt-1 flex gap-2">
                <input
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  required
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="6 位验证码"
                />
                <Button type="button" variant="outline" onClick={handleSendCode} disabled={sendingCode}>
                  {sendingCode ? "发送中" : "发送"}
                </Button>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">完成邮箱验证后会自动登录。</p>
            </div>
          )}

          {error && (
            <p className="text-sm text-red-500">{error}</p>
          )}

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? "处理中..." : isLogin ? "登录" : "注册"}
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          {isLogin ? "没有账户？" : "已有账户？"}
          <button
            onClick={() => { setIsLogin(!isLogin); setError(""); setCode("") }}
            className="ml-1 text-primary hover:underline"
          >
            {isLogin ? "注册" : "登录"}
          </button>
        </p>
      </div>
    </div>
  )
}
