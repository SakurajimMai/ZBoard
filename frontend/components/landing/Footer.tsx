"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { Zap, Send, MessageCircle } from "lucide-react"
import { useI18n } from "@/lib/i18n/context"
import { getPublicSettings } from "@/lib/api"

export default function Footer() {
  const { t } = useI18n()
  const { footer } = t
  const [settings, setSettings] = useState<Record<string, string>>({})

  useEffect(() => {
    getPublicSettings()
      .then((res) => setSettings(res.settings || {}))
      .catch(() => {})
  }, [])

  const tgLink = settings.support_telegram || ""
  const dsLink = settings.support_discord || ""

  return (
    <footer className="border-t border-border bg-card py-12 sm:py-16 px-4 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-7xl">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-8 sm:gap-12 mb-12">
          {/* Brand */}
          <div className="col-span-2 md:col-span-1">
            <div className="flex items-center gap-2.5 mb-4">
              <div className="w-9 h-9 rounded-xl btn-gradient flex items-center justify-center shadow-sm">
                <Zap className="w-5 h-5 text-primary-foreground" strokeWidth={2.5} />
              </div>
              <span className="font-bold text-xl text-foreground">Zboard</span>
            </div>
            <p className="text-sm text-muted-foreground leading-relaxed mb-4">{footer.desc}</p>

            {(tgLink || dsLink) && (
              <div className="flex items-center gap-3">
                {tgLink && (
                  <a
                    href={tgLink}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="w-9 h-9 rounded-xl bg-muted flex items-center justify-center hover:bg-primary/10 hover:text-primary transition-colors"
                    aria-label="Telegram"
                  >
                    <Send className="w-4 h-4" />
                  </a>
                )}
                {dsLink && (
                  <a
                    href={dsLink}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="w-9 h-9 rounded-xl bg-muted flex items-center justify-center hover:bg-primary/10 hover:text-primary transition-colors"
                    aria-label="Discord"
                  >
                    <MessageCircle className="w-4 h-4" />
                  </a>
                )}
              </div>
            )}
          </div>

          {/* Product */}
          <div>
            <h4 className="font-semibold text-foreground text-sm mb-4">{footer.product}</h4>
            <ul className="space-y-3 text-sm text-muted-foreground">
              <li><Link href="#features" className="hover:text-primary transition-colors">{footer.links.features}</Link></li>
              <li><Link href="#pricing"  className="hover:text-primary transition-colors">{footer.links.pricing}</Link></li>
              <li><Link href="#nodes"    className="hover:text-primary transition-colors">{footer.links.nodes}</Link></li>
            </ul>
          </div>

          {/* Support */}
          <div>
            <h4 className="font-semibold text-foreground text-sm mb-4">{footer.support}</h4>
            <ul className="space-y-3 text-sm text-muted-foreground">
              <li><Link href="#"                  className="hover:text-primary transition-colors">{footer.links.docs}</Link></li>
              <li><Link href="/dashboard/ticket"  className="hover:text-primary transition-colors">{footer.links.ticket}</Link></li>
              <li><Link href="#"                  className="hover:text-primary transition-colors">{footer.links.status}</Link></li>
              <li><Link href="#"                  className="hover:text-primary transition-colors">{footer.links.blog}</Link></li>
            </ul>
          </div>

          {/* Legal */}
          <div>
            <h4 className="font-semibold text-foreground text-sm mb-4">{footer.legal}</h4>
            <ul className="space-y-3 text-sm text-muted-foreground">
              <li><Link href="#" className="hover:text-primary transition-colors">{footer.links.terms}</Link></li>
              <li><Link href="#" className="hover:text-primary transition-colors">{footer.links.privacy}</Link></li>
              <li><Link href="#" className="hover:text-primary transition-colors">{footer.links.aup}</Link></li>
            </ul>
          </div>
        </div>

        <div className="border-t border-border pt-8 flex flex-col sm:flex-row items-center justify-between gap-4 text-sm text-muted-foreground">
          <p>© 2025 Zboard. {footer.copyright}.</p>
        </div>
      </div>
    </footer>
  )
}
