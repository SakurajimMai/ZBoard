import type { Metadata } from 'next'
import { cookies, headers } from 'next/headers'
import { Inter, Space_Grotesk } from 'next/font/google'
import { I18nProvider } from '@/lib/i18n/context'
import { ThemeProvider } from '@/components/theme-provider'
import { LOCALES } from '@/lib/i18n/locales'
import { LOCALE_COOKIE_KEY, detectAcceptLanguageLocale, normalizeLocale } from '@/lib/i18n/resolve'
import './globals.css'

const inter = Inter({ subsets: ['latin', 'cyrillic'], variable: '--font-sans', display: 'swap' })
const spaceGrotesk = Space_Grotesk({
  subsets: ['latin'],
  weight: ['500', '600', '700'],
  variable: '--font-display',
  display: 'swap',
})

export const metadata: Metadata = {
  title: 'Zboard — 极速稳定的商业级多端同步加速网络',
  description: 'Zboard 为您提供极速、稳定、安全的全球数据多端同步与协同网络边缘加速服务，采用先进数据加密及高速直连多线路。',
  generator: 'Zboard',
}

function analyticsEnabled() {
  return process.env.NEXT_PUBLIC_VERCEL_ANALYTICS === '1' || process.env.NEXT_PUBLIC_VERCEL_ANALYTICS === 'true'
}

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  const cookieStore = await cookies()
  const requestHeaders = await headers()
  const initialLocale =
    normalizeLocale(cookieStore.get(LOCALE_COOKIE_KEY)?.value) ||
    detectAcceptLanguageLocale(requestHeaders.get('accept-language'))
  const dir = LOCALES.find((l) => l.code === initialLocale)?.dir ?? 'ltr'
  const Analytics = analyticsEnabled()
    ? (await import('@vercel/analytics/next')).Analytics
    : null

  return (
    <html lang={initialLocale} dir={dir} className={`${inter.variable} ${spaceGrotesk.variable} bg-background`} suppressHydrationWarning>
      <body className="font-sans antialiased">
        <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
          <I18nProvider initialLocale={initialLocale}>
            {children}
          </I18nProvider>
        </ThemeProvider>
        {Analytics ? <Analytics /> : null}
      </body>
    </html>
  )
}
