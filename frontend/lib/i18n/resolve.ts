import { type Locale, translations } from "./locales"

export const LEGACY_LOCALE_STORAGE_KEY = "zboard-locale"
export const MANUAL_LOCALE_STORAGE_KEY = "zboard-locale-manual"
export const LOCALE_COOKIE_KEY = "zboard_locale"
export const DEFAULT_LOCALE: Locale = "en"

export function normalizeLocale(value?: string | null): Locale | null {
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

export function detectBrowserLocale(language?: string): Locale {
  return normalizeLocale(language) || DEFAULT_LOCALE
}

export function detectAcceptLanguageLocale(header?: string | null): Locale {
  if (!header) return DEFAULT_LOCALE
  for (const item of header.split(",")) {
    const tag = item.trim().split(";")[0]
    const locale = normalizeLocale(tag)
    if (locale) return locale
  }
  return DEFAULT_LOCALE
}

export function resolvePreferredLocale(options: {
  manual?: string | null
  browser?: string | null
  defaultLanguage?: string | null
}): Locale {
  const manual = normalizeLocale(options.manual)
  if (manual) return manual

  const browser = normalizeLocale(options.browser)
  if (browser) return browser

  return normalizeLocale(options.defaultLanguage) || DEFAULT_LOCALE
}
