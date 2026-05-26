"use client"

import { QRCodeSVG } from "qrcode.react"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { QrCode } from "lucide-react"
import { useI18n } from "@/lib/i18n/context"
import { dashboardCopy } from "@/lib/i18n/dashboard"

interface QRCodeDialogProps {
  url: string
  title?: string
}

export default function QRCodeDialog({ url, title }: QRCodeDialogProps) {
  const { locale } = useI18n()
  const d = dashboardCopy(locale)
  const displayTitle = title || d.overview.subQRTitle

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
          <p className="text-xs text-muted-foreground text-center max-w-[300px] break-all">
            {url}
          </p>
        </div>
      </DialogContent>
    </Dialog>
  )
}
