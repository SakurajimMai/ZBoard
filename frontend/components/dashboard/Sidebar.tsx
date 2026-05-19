"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { useState } from "react"
import {
  LayoutDashboard,
  Globe,
  CreditCard,
  Download,
  Ticket,
  Settings,
  LogOut,
  Zap,
  ChevronRight,
  Menu,
  X,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useI18n } from "@/lib/i18n/context"
import LanguageSwitcher from "@/components/LanguageSwitcher"

export default function Sidebar() {
  const pathname = usePathname()
  const [mobileOpen, setMobileOpen] = useState(false)
  const { t } = useI18n()

  const navItems = [
    { href: "/dashboard",              label: t.dash.overview,      icon: LayoutDashboard },
    { href: "/dashboard/subscription", label: t.dash.subscription,  icon: Globe },
    { href: "/dashboard/billing",      label: t.dash.billing,       icon: CreditCard },
    { href: "/dashboard/download",     label: t.dash.download,      icon: Download },
    { href: "/dashboard/ticket",       label: t.dash.ticket,        icon: Ticket },
    { href: "/dashboard/settings",     label: t.dash.settings,      icon: Settings },
  ]

  const SidebarContent = () => (
    <>
      {/* Logo */}
      <div className="h-16 flex items-center gap-2.5 px-5 border-b border-sidebar-border">
        <Link href="/" className="flex items-center gap-2.5">
          <div className="w-9 h-9 rounded-xl btn-gradient flex items-center justify-center shadow-sm">
            <Zap className="w-5 h-5 text-primary-foreground" strokeWidth={2.5} />
          </div>
          <span className="font-bold text-lg text-sidebar-foreground">Zboard</span>
        </Link>
      </div>

      {/* Nav */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
        {navItems.map((item) => {
          const Icon = item.icon
          const active = pathname === item.href
          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={() => setMobileOpen(false)}
              className={cn(
                "flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all",
                active
                  ? "bg-primary/10 text-primary"
                  : "text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground"
              )}
            >
              <Icon className="w-5 h-5 flex-shrink-0" />
              <span className="flex-1">{item.label}</span>
              {active && <ChevronRight className="w-4 h-4 opacity-60" />}
            </Link>
          )
        })}
      </nav>

      {/* User info + Language switcher */}
      <div className="px-3 py-4 border-t border-sidebar-border space-y-2">
        <div className="flex items-center gap-3 px-4 py-3 rounded-xl bg-sidebar-accent/50">
          <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center text-sm font-bold text-primary">
            用
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-sidebar-foreground truncate">user@example.com</p>
            <p className="text-xs text-sidebar-foreground/50">{t.dash.plan_label}</p>
          </div>
        </div>
        <div className="flex items-center justify-between px-2">
          <LanguageSwitcher align="left" />
          <button className="flex items-center gap-2 px-3 py-2 rounded-xl text-sm text-sidebar-foreground/50 hover:text-destructive hover:bg-destructive/10 transition-colors">
            <LogOut className="w-4 h-4" />
            <span>{t.dash.logout}</span>
          </button>
        </div>
      </div>
    </>
  )

  return (
    <>
      {/* Mobile header */}
      <div className="lg:hidden fixed top-0 left-0 right-0 z-50 h-16 bg-background/95 backdrop-blur-xl border-b border-border flex items-center justify-between px-4">
        <Link href="/" className="flex items-center gap-2.5">
          <div className="w-9 h-9 rounded-xl btn-gradient flex items-center justify-center shadow-sm">
            <Zap className="w-5 h-5 text-primary-foreground" strokeWidth={2.5} />
          </div>
          <span className="font-bold text-lg text-foreground">Zboard</span>
        </Link>
        <div className="flex items-center gap-2">
          <LanguageSwitcher align="right" />
          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            className="p-2 rounded-xl hover:bg-accent transition-colors"
            aria-label={mobileOpen ? "关闭菜单" : "打开菜单"}
          >
            {mobileOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
        </div>
      </div>

      {/* Mobile sidebar overlay */}
      {mobileOpen && (
        <div
          className="lg:hidden fixed inset-0 z-40 bg-background/80 backdrop-blur-sm"
          onClick={() => setMobileOpen(false)}
        />
      )}

      {/* Mobile sidebar */}
      <aside
        className={cn(
          "lg:hidden fixed top-0 left-0 z-50 w-72 h-full bg-sidebar border-r border-sidebar-border flex flex-col transform transition-transform duration-300 ease-in-out",
          mobileOpen ? "translate-x-0" : "-translate-x-full"
        )}
      >
        <SidebarContent />
      </aside>

      {/* Desktop sidebar */}
      <aside className="hidden lg:flex w-64 flex-shrink-0 border-r border-sidebar-border bg-sidebar min-h-screen flex-col sticky top-0">
        <SidebarContent />
      </aside>
    </>
  )
}
