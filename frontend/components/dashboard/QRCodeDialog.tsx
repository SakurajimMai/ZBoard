"use client"

import { QRCodeSVG } from "qrcode.react"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { QrCode } from "lucide-react"

interface QRCodeDialogProps {
  url: string
  title?: string
}

export default function QRCodeDialog({ url, title = "同步配置二维码" }: QRCodeDialogProps) {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <QrCode className="w-4 h-4 mr-1.5" /> 生成二维码
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
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
