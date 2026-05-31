"use client"

import { useEffect, useRef } from "react"

declare global {
  interface Window {
    turnstile?: {
      render: (el: HTMLElement, opts: Record<string, unknown>) => string
      remove: (id: string) => void
      reset: (id: string) => void
      execute?: (id: string) => void
    }
    grecaptcha?: {
      render: (el: HTMLElement, opts: Record<string, unknown>) => number
      reset: (id: number) => void
    }
    hcaptcha?: {
      render: (el: HTMLElement, opts: Record<string, unknown>) => string
      remove: (id: string) => void
      reset: (id: string) => void
    }
  }
}

type Provider = "none" | "turnstile" | "recaptcha" | "hcaptcha"

export interface CaptchaProps {
  provider: Provider
  siteKey: string
  mode?: "managed" | "non-interactive" | "invisible"
  onToken: (token: string) => void
  onError?: (msg: string) => void
}

const SCRIPTS: Record<Exclude<Provider, "none">, { src: string; id: string }> = {
  turnstile: {
    src: "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit",
    id: "captcha-script-turnstile",
  },
  recaptcha: {
    src: "https://www.google.com/recaptcha/api.js?render=explicit",
    id: "captcha-script-recaptcha",
  },
  hcaptcha: {
    src: "https://hcaptcha.com/1/api.js?render=explicit",
    id: "captcha-script-hcaptcha",
  },
}

function loadScript(provider: Exclude<Provider, "none">): Promise<void> {
  const cfg = SCRIPTS[provider]
  if (typeof document === "undefined") return Promise.resolve()
  if (document.getElementById(cfg.id)) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const s = document.createElement("script")
    s.id = cfg.id
    s.src = cfg.src
    s.async = true
    s.defer = true
    s.onload = () => resolve()
    s.onerror = () => reject(new Error(`脚本加载失败: ${provider}`))
    document.head.appendChild(s)
  })
}

async function waitFor<T>(get: () => T | undefined, timeoutMs = 10000): Promise<T> {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    const v = get()
    if (v) return v
    await new Promise((r) => setTimeout(r, 80))
  }
  throw new Error("人机验证脚本超时")
}

export function Captcha({ provider, siteKey, mode = "managed", onToken, onError }: CaptchaProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const widgetIdRef = useRef<string | number | null>(null)

  // 把回调存进 ref 并每次渲染刷新:调用方(登录/注册表单)往往传内联箭头函数,
  // 每次按键都生成新引用。若把 onToken/onError 放进 effect 依赖,widget 会被反复
  // 销毁重建 —— 这正是 Turnstile 报 "preloaded but not used" / 401 / postMessage
  // origin 不匹配的根因。用 ref 让 widget 只在 provider/siteKey/mode 真正变化时重建。
  const onTokenRef = useRef(onToken)
  const onErrorRef = useRef(onError)
  useEffect(() => {
    onTokenRef.current = onToken
    onErrorRef.current = onError
  })

  useEffect(() => {
    if (provider === "none" || !siteKey) return
    let cancelled = false

    ;(async () => {
      try {
        await loadScript(provider)
        if (cancelled || !containerRef.current) return

        if (provider === "turnstile") {
          const ts = await waitFor(() => window.turnstile)
          if (cancelled || !containerRef.current) return
          containerRef.current.innerHTML = ""
          widgetIdRef.current = ts.render(containerRef.current, {
            sitekey: siteKey,
            execution: mode === "invisible" ? "execute" : undefined,
            size: mode === "invisible" ? "invisible" : undefined,
            appearance: mode === "non-interactive" ? "interaction-only" : "always",
            callback: (t: string) => onTokenRef.current(t),
            "error-callback": () => onErrorRef.current?.("人机验证出错"),
            "expired-callback": () => onTokenRef.current(""),
          })
          if (mode === "invisible") {
            ts.execute?.(widgetIdRef.current)
          }
        } else if (provider === "recaptcha") {
          const rc = await waitFor(() => window.grecaptcha)
          if (cancelled || !containerRef.current) return
          containerRef.current.innerHTML = ""
          widgetIdRef.current = rc.render(containerRef.current, {
            sitekey: siteKey,
            callback: (t: string) => onTokenRef.current(t),
            "expired-callback": () => onTokenRef.current(""),
            "error-callback": () => onErrorRef.current?.("人机验证出错"),
          })
        } else if (provider === "hcaptcha") {
          const hc = await waitFor(() => window.hcaptcha)
          if (cancelled || !containerRef.current) return
          containerRef.current.innerHTML = ""
          widgetIdRef.current = hc.render(containerRef.current, {
            sitekey: siteKey,
            callback: (t: string) => onTokenRef.current(t),
            "expired-callback": () => onTokenRef.current(""),
            "error-callback": () => onErrorRef.current?.("人机验证出错"),
          })
        }
      } catch (err) {
        if (!cancelled) onErrorRef.current?.(err instanceof Error ? err.message : "人机验证加载失败")
      }
    })()

    return () => {
      cancelled = true
      try {
        if (provider === "turnstile" && typeof widgetIdRef.current === "string") {
          window.turnstile?.remove(widgetIdRef.current)
        } else if (provider === "hcaptcha" && typeof widgetIdRef.current === "string") {
          window.hcaptcha?.remove(widgetIdRef.current)
        }
      } catch {}
      widgetIdRef.current = null
    }
  }, [provider, siteKey, mode])

  if (provider === "none" || !siteKey) return null
  return <div ref={containerRef} className="flex justify-center" />
}

export function captchaEnabled(settings: Record<string, string> | undefined, scene: "register" | "login" | "forgot" | "ticket"): boolean {
  if (!settings) return false
  const provider = settings.captcha_provider
  if (!provider || provider === "none") return false
  const key = settings.captcha_site_key
  if (!key) return false
  const flag = settings[`captcha_enabled_${scene}`]
  return flag === "1" || flag === "true"
}
