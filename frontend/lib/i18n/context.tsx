"use client"

import { createContext, useContext, useEffect, useState, type ReactNode } from "react"
import { getPublicSettings } from "@/lib/api"
import { type Locale, type TranslationDict, translations, LOCALES } from "./locales"
import {
  DEFAULT_LOCALE,
  LEGACY_LOCALE_STORAGE_KEY,
  LOCALE_COOKIE_KEY,
  MANUAL_LOCALE_STORAGE_KEY,
  normalizeLocale,
  resolvePreferredLocale,
} from "./resolve"

interface I18nContextValue {
  locale: Locale
  t: TranslationDict
  setLocale: (locale: Locale) => void
  dir: "ltr" | "rtl"
}

const I18nContext = createContext<I18nContextValue | null>(null)

function resolveClientLocale(defaultLanguage?: string | null): Locale {
  if (typeof window === "undefined") return normalizeLocale(defaultLanguage) || DEFAULT_LOCALE
  return resolvePreferredLocale({
    manual: localStorage.getItem(MANUAL_LOCALE_STORAGE_KEY),
    browser: navigator.language,
    defaultLanguage,
  })
}

function persistLocaleCookie(locale: Locale) {
  if (typeof document === "undefined") return
  document.cookie = `${LOCALE_COOKIE_KEY}=${encodeURIComponent(locale)}; Path=/; Max-Age=31536000; SameSite=Lax`
}

export function I18nProvider({ children, initialLocale = DEFAULT_LOCALE }: { children: ReactNode; initialLocale?: Locale }) {
  const [locale, setLocaleState] = useState<Locale>(initialLocale)

  useEffect(() => {
    let cancelled = false
    const firstLocale = resolveClientLocale()
    setLocaleState(firstLocale)
    if (normalizeLocale(localStorage.getItem(MANUAL_LOCALE_STORAGE_KEY))) {
      persistLocaleCookie(firstLocale)
    }
    getPublicSettings()
      .then((res) => {
        if (!cancelled) setLocaleState(resolveClientLocale(res.settings?.default_language))
      })
      .catch(() => {
        if (!cancelled) setLocaleState(firstLocale)
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
      localStorage.setItem(MANUAL_LOCALE_STORAGE_KEY, next)
      localStorage.removeItem(LEGACY_LOCALE_STORAGE_KEY)
      persistLocaleCookie(next)
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
