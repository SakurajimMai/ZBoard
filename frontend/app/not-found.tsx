import Link from "next/link"
import { Compass, ArrowLeft, Home } from "lucide-react"
import { Button } from "@/components/ui/button"

export default function NotFound() {
  return (
    <main className="min-h-dvh flex items-center justify-center bg-background px-4 py-16">
      <div className="w-full max-w-md text-center">
        {/* Decorative icon */}
        <div className="relative mx-auto mb-8 w-24 h-24">
          <div className="absolute inset-0 rounded-full bg-primary/10 blur-2xl" />
          <div className="relative w-24 h-24 rounded-2xl bg-card border border-border flex items-center justify-center shadow-sm">
            <Compass className="w-12 h-12 text-primary" strokeWidth={1.5} aria-hidden="true" />
          </div>
        </div>

        {/* Status code */}
        <p
          className="text-7xl sm:text-8xl font-bold tracking-tight bg-gradient-to-br from-primary to-primary/40 bg-clip-text text-transparent leading-none"
          aria-label="404"
        >
          404
        </p>

        {/* Headline */}
        <h1 className="mt-6 text-2xl sm:text-3xl font-semibold text-foreground">
          页面走丢了
        </h1>

        {/* Description */}
        <p className="mt-3 text-sm text-muted-foreground leading-relaxed">
          您访问的页面不存在或已被移除。
          <br className="hidden sm:block" />
          请检查链接是否正确，或返回首页继续探索。
        </p>

        {/* Actions */}
        <div className="mt-8 flex flex-col sm:flex-row items-center justify-center gap-3">
          <Link href="/" className="w-full sm:w-auto">
            <Button className="w-full gap-2" size="lg">
              <Home className="w-4 h-4" aria-hidden="true" />
              返回首页
            </Button>
          </Link>
          <Link href="/dashboard" className="w-full sm:w-auto">
            <Button variant="outline" className="w-full gap-2" size="lg">
              <ArrowLeft className="w-4 h-4" aria-hidden="true" />
              进入控制台
            </Button>
          </Link>
        </div>

        {/* Helper links */}
        <p className="mt-10 text-xs text-muted-foreground">
          如需帮助，请{" "}
          <Link href="/dashboard/ticket" className="text-primary hover:underline">
            提交工单
          </Link>
          {" 或 "}
          <Link href="/login" className="text-primary hover:underline">
            登录账户
          </Link>
        </p>
      </div>
    </main>
  )
}
