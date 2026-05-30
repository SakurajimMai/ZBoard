"use client"

import { useState } from "react"
import { QRCodeSVG } from "qrcode.react"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { QrCode, Copy, Check } from "lucide-react"
import { useI18n } from "@/lib/i18n/context"
import { dashboardCopy } from "@/lib/i18n/dashboard"

interface QRCodeDialogProps {
  url: string
  title?: string
}

// maskSubscriptionURL hides the secret subscription token in the visible text
// while keeping enough of the URL recognizable. The token is the path segment
// after `/sub/`; we show only its last 4 chars. The QR code and the copy button
// still carry the full URL — we just don't render the raw token where it could
// be shoulder-surfed, screenshotted, or indexed.
function maskSubscriptionURL(raw: string): string {
  try {
    const u = new URL(raw)
    const marker = "/sub/"
    const idx = u.pathname.indexOf(marker)
    if (idx === -1) return raw
    const before = u.pathname.slice(0, idx + marker.length)
    const token = u.pathname.slice(idx + marker.length)
    if (token.length <= 4) return raw
    const masked = "****" + token.slice(-4)
    u.pathname = before + masked
    return u.toString()
  } catch {
    return raw
  }
}

export default function QRCodeDialog({ url, title }: QRCodeDialogProps) {
  const { locale } = useI18n()
  const d = dashboardCopy(locale)
  const displayTitle = title || d.overview.subQRTitle
  const [copied, setCopied] = useState(false)

  const handleCopy = () => {
    navigator.clipboard.writeText(url)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <QrCode className="w-4 h-4 mr-1.5" /> {d.overview.generateQR}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{displayTitle}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col items-center gap-4 py-4">
          <div className="rounded-xl border bg-white p-4">
            <QRCodeSVG
              value={url}
              size={240}
              level="M"
              includeMargin={false}
            />
          </div>
          <p className="text-xs text-muted-foreground text-center max-w-[300px] break-all font-mono">
            {maskSubscriptionURL(url)}
          </p>
          <Button variant="outline" size="sm" onClick={handleCopy}>
            {copied ? (
              <><Check className="w-4 h-4 mr-1.5" /> {d.common.copied}</>
            ) : (
              <><Copy className="w-4 h-4 mr-1.5" /> {d.common.copy}</>
            )}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
