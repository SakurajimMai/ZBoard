import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import { Analytics } from '@vercel/analytics/next'
import { I18nProvider } from '@/lib/i18n/context'
import './globals.css'

const inter = Inter({ subsets: ['latin', 'cyrillic'], variable: '--font-sans' })

export const metadata: Metadata = {
  title: 'Zboard — 高速稳定的商业订阅机场',
  description: 'Zboard 提供高速、稳定、安全的全球网络访问服务，支持多种协议，覆盖数十个国家节点。',
  generator: 'Zboard',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="zh-CN" className={`${inter.variable} bg-background`}>
      <body className="font-sans antialiased">
        <I18nProvider>
          {children}
        </I18nProvider>
        {process.env.NODE_ENV === 'production' && <Analytics />}
      </body>
    </html>
  )
}
