import type { Metadata, Viewport } from 'next'
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

// 移动端视口:width=device-width + initial-scale=1 让页面按设备真实宽度渲染,
// 否则手机会用 ~980px 桌面视口缩小整页。不设 viewportFit:cover —— 本项目是
// 浏览器网页(非独立 PWA),固定头部用的是普通 top-0、没有铺 safe-area padding,
// 默认 auto 能保证内容不被刘海/状态栏遮挡。不限制 maximumScale,保留无障碍缩放。
export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#f8fafc' },
    { media: '(prefers-color-scheme: dark)', color: '#0d0f1a' },
  ],
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
