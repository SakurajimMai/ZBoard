"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { login, register, ApiError } from "@/lib/api"
import { Button } from "@/components/ui/button"

export default function LoginPage() {
  const router = useRouter()
  const [isLogin, setIsLogin] = useState(true)
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    setLoading(true)
    try {
      if (isLogin) {
        await login(email, password)
        router.push("/dashboard")
      } else {
        await register(email, password)
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
            onClick={() => { setIsLogin(!isLogin); setError("") }}
            className="ml-1 text-primary hover:underline"
          >
            {isLogin ? "注册" : "登录"}
          </button>
        </p>

        <p className="text-center text-sm text-muted-foreground">
          <a href="/admin/login" className="text-primary hover:underline">管理员入口</a>
        </p>
      </div>
    </div>
  )
}
