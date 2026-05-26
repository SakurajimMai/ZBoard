"use client"

import { createContext, useContext, useEffect, useState, type ReactNode } from "react"
import { getPublicSettings } from "@/lib/api"
import { type Locale, type TranslationDict, translations, LOCALES } from "./locales"

interface I18nContextValue {
  locale: Locale
  t: TranslationDict
  setLocale: (locale: Locale) => void
  dir: "ltr" | "rtl"
}

const I18nContext = createContext<I18nContextValue | null>(null)

const LEGACY_STORAGE_KEY = "zboard-locale"
const MANUAL_STORAGE_KEY = "zboard-locale-manual"
const DEFAULT_LOCALE: Locale = "en"

function normalizeLocale(value?: string | null): Locale | null {
  if (!value || value === "auto") return null
  if (value in translations) return value as Locale
  if (value.startsWith("zh-TW") || value.startsWith("zh-HK")) return "zh-TW"
  if (value.startsWith("zh")) return "zh-CN"
  if (value.startsWith("ja")) return "ja"
  if (value.startsWith("ko")) return "ko"
  if (value.startsWith("vi")) return "vi"
  if (value.startsWith("fa")) return "fa"
  if (value.startsWith("ru")) return "ru"
  if (value.startsWith("en")) return "en"
  return null
}

function detectBrowserLocale(): Locale {
  if (typeof window === "undefined") return DEFAULT_LOCALE
  const browser = navigator.language
  if (browser.startsWith("zh-TW") || browser.startsWith("zh-HK")) return "zh-TW"
  if (browser.startsWith("zh")) return "zh-CN"
  if (browser.startsWith("ja")) return "ja"
  if (browser.startsWith("ko")) return "ko"
  if (browser.startsWith("vi")) return "vi"
  if (browser.startsWith("fa")) return "fa"
  if (browser.startsWith("ru")) return "ru"
  if (browser.startsWith("en")) return "en"
  return DEFAULT_LOCALE
}

function resolvePreferredLocale(defaultLanguage?: string): Locale {
  if (typeof window === "undefined") return normalizeLocale(defaultLanguage) || DEFAULT_LOCALE
  const manual = normalizeLocale(localStorage.getItem(MANUAL_STORAGE_KEY))
  if (manual) return manual
  const browser = detectBrowserLocale()
  if (browser) return browser
  return normalizeLocale(defaultLanguage) || DEFAULT_LOCALE
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(DEFAULT_LOCALE)

  useEffect(() => {
    let cancelled = false
    setLocaleState(resolvePreferredLocale())
    getPublicSettings()
      .then((res) => {
        if (!cancelled) setLocaleState(resolvePreferredLocale(res.settings?.default_language))
      })
      .catch(() => {
        if (!cancelled) setLocaleState(resolvePreferredLocale())
      })
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    const localeInfo = LOCALES.find((l) => l.code === locale)
    document.documentElement.lang = locale
    document.documentElement.dir = localeInfo?.dir ?? "ltr"
  }, [locale])

  const setLocale = (next: Locale) => {
    if (typeof window !== "undefined") {
      localStorage.setItem(MANUAL_STORAGE_KEY, next)
      localStorage.removeItem(LEGACY_STORAGE_KEY)
    }
    setLocaleState(next)
  }

  return (
    <I18nContext.Provider
      value={{
        locale,
        t: translations[locale],
        setLocale,
        dir: LOCALES.find((l) => l.code === locale)?.dir === "rtl" ? "rtl" : "ltr",
      }}
    >
      {children}
    </I18nContext.Provider>
  )
}

export function useI18n() {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error("useI18n must be used inside <I18nProvider>")
  return ctx
}
