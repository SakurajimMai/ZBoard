"use client"

import { useRef, useState, useEffect } from "react"
import { Globe, Check } from "lucide-react"
import { LOCALES, type Locale } from "@/lib/i18n/locales"
import { useI18n } from "@/lib/i18n/context"
import { cn } from "@/lib/utils"

export default function LanguageSwitcher({ align = "right" }: { align?: "left" | "right" }) {
  const { locale, setLocale } = useI18n()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const current = LOCALES.find((l) => l.code === locale) ?? LOCALES[0]

  // Close on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener("mousedown", handleClick)
    return () => document.removeEventListener("mousedown", handleClick)
  }, [open])

  function handleSelect(code: Locale) {
    setLocale(code)
    setOpen(false)
  }

  return (
    <div ref={ref} className="relative">
      {/* Trigger */}
      <button
        onClick={() => setOpen((v) => !v)}
        className={cn(
          "flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm font-medium transition-colors",
          "text-muted-foreground hover:text-foreground hover:bg-accent",
          open && "bg-accent text-foreground"
        )}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label="切换语言"
      >
        <Globe className="w-4 h-4 flex-shrink-0" />
        <span className="hidden sm:inline max-w-[80px] truncate">{current.nativeLabel}</span>
      </button>

      {/* Dropdown */}
      {open && (
        <div
          role="listbox"
          aria-label="选择语言"
          className={cn(
            "absolute z-[200] top-full mt-1.5 w-44 rounded-xl border border-border bg-background shadow-lg shadow-black/5 overflow-hidden animate-in fade-in-0 zoom-in-95 duration-150",
            align === "right" ? "right-0" : "left-0"
          )}
        >
          {/* List */}
          <ul className="py-1">
            {LOCALES.map((lang) => {
              const isActive = locale === lang.code
              return (
                <li key={lang.code}>
                  <button
                    role="option"
                    aria-selected={isActive}
                    onClick={() => handleSelect(lang.code)}
                    className={cn(
                      "w-full flex items-center justify-between gap-2 px-4 py-2.5 text-sm transition-colors text-left",
                      isActive
                        ? "text-foreground font-medium bg-accent/60"
                        : "text-muted-foreground hover:text-foreground hover:bg-accent/40"
                    )}
                  >
                    <span>{lang.nativeLabel}</span>
                    {isActive && <Check className="w-4 h-4 text-primary flex-shrink-0" />}
                  </button>
                </li>
              )
            })}
          </ul>

          {/* Footer — mirrors the design in the screenshot */}
          <div className="border-t border-border px-4 py-2.5 flex items-center gap-2 bg-muted/30">
            <Globe className="w-3.5 h-3.5 text-muted-foreground flex-shrink-0" />
            <span className="text-xs text-muted-foreground truncate">{current.nativeLabel}</span>
          </div>
        </div>
      )}
    </div>
  )
}
