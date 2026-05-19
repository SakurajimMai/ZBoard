"use client"

import { createContext, useContext, useEffect, useState, type ReactNode } from "react"
import { type Locale, type TranslationDict, translations, LOCALES } from "./locales"

interface I18nContextValue {
  locale: Locale
  t: TranslationDict
  setLocale: (locale: Locale) => void
  dir: "ltr" | "rtl"
}

const I18nContext = createContext<I18nContextValue | null>(null)

const STORAGE_KEY = "zboard-locale"
const DEFAULT_LOCALE: Locale = "zh-CN"

function detectLocale(): Locale {
  if (typeof window === "undefined") return DEFAULT_LOCALE
  const stored = localStorage.getItem(STORAGE_KEY) as Locale | null
  if (stored && translations[stored]) return stored
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

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(DEFAULT_LOCALE)

  useEffect(() => {
    setLocaleState(detectLocale())
  }, [])

  useEffect(() => {
    const localeInfo = LOCALES.find((l) => l.code === locale)
    document.documentElement.lang = locale
    document.documentElement.dir = localeInfo?.dir ?? "ltr"
    localStorage.setItem(STORAGE_KEY, locale)
  }, [locale])

  const setLocale = (next: Locale) => setLocaleState(next)

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
