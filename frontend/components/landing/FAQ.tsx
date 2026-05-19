"use client"

import { useState } from "react"
import { ChevronDown, HelpCircle } from "lucide-react"
import { useI18n } from "@/lib/i18n/context"

export default function FAQ() {
  const [open, setOpen] = useState<number | null>(0)
  const { t } = useI18n()

  return (
    <section id="faq" className="py-20 sm:py-28 px-4 sm:px-6 lg:px-8 bg-background">
      <div className="mx-auto max-w-3xl">
        <div className="text-center mb-12 sm:mb-14">
          <div className="inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/5 px-4 py-2 text-sm text-primary font-medium mb-4">
            <HelpCircle className="w-4 h-4" />
            {t.faq.title}
          </div>
          <h2 className="text-balance text-3xl sm:text-4xl md:text-5xl font-bold text-foreground mb-3">
            {t.faq.title}
          </h2>
          <p className="text-muted-foreground text-base sm:text-lg">{t.faq.subtitle}</p>
        </div>

        <div className="space-y-3">
          {t.faq.items.map((faq, i) => (
            <div
              key={i}
              className={`rounded-2xl border bg-card overflow-hidden transition-all card-shadow ${
                open === i ? "border-primary/30" : "border-border/50"
              }`}
            >
              <button
                className="w-full flex items-center justify-between px-5 sm:px-6 py-4 sm:py-5 text-left hover:bg-accent/50 transition-colors"
                onClick={() => setOpen(open === i ? null : i)}
                aria-expanded={open === i}
              >
                <span className="font-medium text-foreground pr-4 text-sm sm:text-base">{faq.q}</span>
                <div className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 transition-colors ${
                  open === i ? "bg-primary/10" : "bg-muted"
                }`}>
                  <ChevronDown
                    className={`w-4 h-4 transition-transform duration-200 ${
                      open === i ? "rotate-180 text-primary" : "text-muted-foreground"
                    }`}
                  />
                </div>
              </button>
              {open === i && (
                <div className="px-5 sm:px-6 pb-5 sm:pb-6 animate-in slide-in-from-top-1 duration-200">
                  <p className="text-sm sm:text-base text-muted-foreground leading-relaxed">{faq.a}</p>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
