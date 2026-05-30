"use client"

import { useEffect, useState } from "react"
import { Monitor, Moon, Sun } from "lucide-react"
import { useTheme } from "next-themes"
import { cn } from "@/lib/utils"

// 三态循环:跟随系统 → 浅色 → 深色 → 跟随系统。图标始终反映当前生效主题,
// 标签说明「点击后会切到哪个状态」,方便无障碍朗读。
const ORDER = ["system", "light", "dark"] as const
type ThemeValue = (typeof ORDER)[number]

const LABEL: Record<ThemeValue, string> = {
  system: "跟随系统",
  light: "浅色",
  dark: "深色",
}

export default function ThemeToggle({ className }: { className?: string }) {
  const { theme, resolvedTheme, setTheme } = useTheme()
  const [mounted, setMounted] = useState(false)

  // next-themes 在客户端 hydration 后才知道真实主题;挂载前渲染占位避免水合不一致。
  useEffect(() => setMounted(true), [])

  if (!mounted) {
    return (
      <button
        type="button"
        aria-label="切换主题"
        className={cn(
          "flex items-center justify-center size-9 rounded-lg text-muted-foreground",
          className,
        )}
      >
        <Sun className="w-4 h-4" />
      </button>
    )
  }

  const current = (theme ?? "system") as ThemeValue
  const next = ORDER[(ORDER.indexOf(current) + 1) % ORDER.length]
  // 图标反映「当前实际显示」的明暗,system 态用显示器图标提示是自动模式。
  const Icon = current === "system" ? Monitor : resolvedTheme === "dark" ? Moon : Sun

  return (
    <button
      type="button"
      onClick={() => setTheme(next)}
      title={`主题:${LABEL[current]}(点击切换到${LABEL[next]})`}
      aria-label={`当前主题 ${LABEL[current]},点击切换到 ${LABEL[next]}`}
      className={cn(
        "flex items-center justify-center size-9 rounded-lg transition-colors",
        "text-muted-foreground hover:text-foreground hover:bg-accent",
        className,
      )}
    >
      <Icon className="w-4 h-4" />
    </button>
  )
}
