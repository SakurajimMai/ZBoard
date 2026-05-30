"use client"

import { useEffect, useState } from "react"
import {
  Copy, RefreshCw, Server, Shield,
  ChevronDown, Info, CreditCard, AlertCircle, CheckCircle2
} from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  getMe, getSubscription, resetSubscriptionToken,
  getPublicSettings,
  resetMyTraffic, resetMyUUID,
  getPaymentMethods, payOrder, getPlans, createOrder,
  buildSubscriptionUrl,
  buildSubscriptionUrlFromBase,
} from "@/lib/api"
import QRCodeDialog from "@/components/dashboard/QRCodeDialog"
import { useI18n } from "@/lib/i18n/context"
import { dashboardCopy } from "@/lib/i18n/dashboard"

function generateSubId(token: string): string {
  if (!token) return "LINK-PENDING"
  const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
  let seed = 0
  for (let i = 0; i < token.length; i++) {
    seed = (seed * 31 + token.charCodeAt(i)) >>> 0
  }
  let out = ""
  for (let i = 0; i < 6; i++) {
    seed = (seed * 1664525 + 1013904223) >>> 0
    out += alphabet[seed % alphabet.length]
  }
  return `LINK-${out.slice(0, 3)}-${out.slice(3)}`
}

function formatBytes(bytes: number): { value: string; unit: string } {
  if (bytes >= 1073741824) return { value: (bytes / 1073741824).toFixed(2), unit: "GB" }
  if (bytes >= 1048576) return { value: (bytes / 1048576).toFixed(2), unit: "MB" }
  if (bytes >= 1024) return { value: (bytes / 1024).toFixed(2), unit: "KB" }
  return { value: bytes.toString(), unit: "B" }
}

function bytesToGB(value: number | null | undefined) {
  return String(Number(((value || 0) / 1073741824).toFixed(1)).toString())
}

type BillingPeriod = "monthly" | "quarterly" | "yearly"

function periodPrice(plan: any, period: BillingPeriod) {
  const monthly = Number(plan.price || 0)
  if (period === "quarterly") {
    const quarterly = Number(plan.quarterly_price || 0)
    return (quarterly > 0 ? quarterly : monthly * 3).toFixed(2)
  }
  if (period === "yearly") {
    const yearly = Number(plan.yearly_price || 0)
    return (yearly > 0 ? yearly : monthly * 12).toFixed(2)
  }
  return monthly.toFixed(2)
}

export default function Overview() {
  const { locale } = useI18n()
  const d = dashboardCopy(locale)

  const billingPeriods: { value: BillingPeriod; label: string }[] = [
    { value: "monthly", label: d.overview.monthly },
    { value: "quarterly", label: d.overview.quarterly },
    { value: "yearly", label: d.overview.yearly },
  ]
  const clientOptions = [
    { value: "general", label: d.overview.clientGeneral },
    { value: "v2rayn", label: "v2rayN / v2rayNG", target: "v2rayn" },
    { value: "shadowrocket", label: "Shadowrocket", target: "shadowrocket" },
    { value: "passwall", label: "Passwall", target: "passwall" },
    { value: "clash", label: "Clash / Mihomo", target: "clash" },
    { value: "singbox", label: d.overview.clientSingbox, target: "sing-box" },
    { value: "hiddify", label: "Hiddify", target: "hiddify" },
    { value: "furious", label: "Furious", target: "furious" },
  ]

  const [user, setUser] = useState<any>(null)
  const [plans, setPlans] = useState<any[]>([])
  const [subToken, setSubToken] = useState("")
  const [settings, setSettings] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [copied, setCopied] = useState<string | null>(null)
  const [clientType, setClientType] = useState("general")
  const [resettingTraffic, setResettingTraffic] = useState(false)
  const [resettingUUID, setResettingUUID] = useState(false)
  const [buyingPlanKey, setBuyingPlanKey] = useState<string | null>(null)
  const [trafficDialogOpen, setTrafficDialogOpen] = useState(false)
  const [uuidDialogOpen, setUUIDDialogOpen] = useState(false)
  const [notice, setNotice] = useState<{ type: "success" | "error"; message: string } | null>(null)

  useEffect(() => {
    Promise.all([
      getMe(),
      getSubscription(),
      getPlans().catch(() => ({ items: [] })),
      getPublicSettings().catch(() => ({ settings: {} })),
    ])
      .then(([meRes, subRes, plansRes, settingsRes]) => {
        setUser(meRes.user)
        setSubToken(subRes.token)
        setPlans(plansRes.items || [])
        setSettings(settingsRes.settings)
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="text-muted-foreground p-8">{d.common.loading}</div>
  if (!user) return <div className="text-destructive p-8">{d.common.loadError}</div>

  const target = clientOptions.find((item) => item.value === clientType)?.target
  const mainLink = buildSubscriptionUrl(subToken, target, settings)

  const backupDomain = settings.backup_subscription_domain || ""
  const backupLink = backupDomain ? buildSubscriptionUrlFromBase(backupDomain, subToken, target) : ""

  const totalBytes = user.traffic_limit || 0
  const usedBytes = user.traffic_used || 0
  const totalFormatted = formatBytes(totalBytes)
  const usedFormatted = formatBytes(usedBytes)
  const usedPct = totalBytes > 0 ? Math.min((usedBytes / totalBytes) * 100, 100) : 0

  const subId = generateSubId(subToken)
  const expireDate = user.expired_at ? new Date(user.expired_at).toLocaleDateString(locale) : d.common.none

  const handleCopy = (text: string, id: string) => {
    navigator.clipboard.writeText(text)
    setCopied(id)
    setTimeout(() => setCopied(null), 2000)
  }

  // redirectToPayment navigates to a gateway pay URL only after confirming it's
  // an http(s) absolute URL. pay_url comes from our own backend providers, but
  // validating the scheme here blocks an open-redirect / javascript: payload if
  // a provider is ever misconfigured or compromised.
  const redirectToPayment = (rawURL: string): boolean => {
    try {
      const u = new URL(rawURL)
      if (u.protocol !== "https:" && u.protocol !== "http:") {
        return false
      }
      window.location.href = u.toString()
      return true
    } catch {
      return false
    }
  }

  const handleResetToken = async () => {
    try {
      const res = await resetSubscriptionToken()
      setSubToken(res.token)
    } catch (e) {
      console.error(e)
    }
  }

  const resetPriceRaw = (user as any)?.reset_traffic_price ?? "0"
  const resetPriceNum = Number(resetPriceRaw) || 0
  const resetPriceEnabled = resetPriceNum > 0

  const openTrafficResetDialog = () => {
    if (!resetPriceEnabled) {
      setNotice({ type: "error", message: d.overview.trafficNotAvailable })
      return
    }
    setTrafficDialogOpen(true)
  }

  const handleResetTrafficPayment = async () => {
    if (resettingTraffic) return
    if (!resetPriceEnabled) {
      setNotice({ type: "error", message: d.overview.trafficNotAvailable })
      return
    }
    setResettingTraffic(true)
    setNotice(null)
    try {
      const methodsRes = await getPaymentMethods()
      const method = methodsRes.methods?.[0]
      if (!method?.name) {
        throw new Error(d.overview.paymentMethodMissing)
      }
      const orderRes = await resetMyTraffic()
      const orderNo = orderRes.order?.order_no
      if (!orderNo) {
        throw new Error(d.overview.orderCreateFailed)
      }
      const payType = method.provider_type === "epay" ? "alipay" : undefined
      const payRes = await payOrder(orderNo, method.name, payType)
      if (!payRes.pay_url) {
        throw new Error(d.overview.payGatewayError)
      }
      if (!redirectToPayment(payRes.pay_url)) {
        throw new Error(d.overview.payGatewayError)
      }
    } catch (e: any) {
      setNotice({ type: "error", message: e?.message || d.overview.createOrderFailed })
    } finally {
      setResettingTraffic(false)
    }
  }

  const handleResetUUID = async () => {
    if (resettingUUID) return
    setResettingUUID(true)
    setNotice(null)
    try {
      await resetMyUUID()
      setUUIDDialogOpen(false)
      setNotice({ type: "success", message: d.overview.uuidResetSuccess })
    } catch (e: any) {
      setNotice({ type: "error", message: e?.message || d.overview.uuidResetFailed })
    } finally {
      setResettingUUID(false)
    }
  }

  const handleBuyPlan = async (plan: any, period: BillingPeriod) => {
    const key = `${plan.id}:${period}`
    if (buyingPlanKey) return
    setBuyingPlanKey(key)
    setNotice(null)
    try {
      const methodsRes = await getPaymentMethods()
      const method = methodsRes.methods?.[0]
      if (!method?.name) {
        throw new Error(d.overview.paymentMethodMissing)
      }
      const orderRes = await createOrder(Number(plan.id), period)
      const orderNo = orderRes.order?.order_no
      if (!orderNo) {
        throw new Error(d.overview.orderCreateFailed)
      }
      const payType = method.provider_type === "epay" ? "alipay" : undefined
      const payRes = await payOrder(orderNo, method.name, payType)
      if (!payRes.pay_url) {
        throw new Error(d.overview.payGatewayError)
      }
      if (!redirectToPayment(payRes.pay_url)) {
        throw new Error(d.overview.payGatewayError)
      }
    } catch (e: any) {
      setNotice({ type: "error", message: e?.message || d.overview.createOrderFailed })
    } finally {
      setBuyingPlanKey(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="space-y-6">
          {notice && (
            <div className={`rounded-lg border px-4 py-3 text-sm flex items-start gap-2 ${
              notice.type === "success"
                ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
                : "border-destructive/30 bg-destructive/10 text-destructive"
            }`}>
              {notice.type === "success" ? <CheckCircle2 className="w-4 h-4 mt-0.5" /> : <AlertCircle className="w-4 h-4 mt-0.5" />}
              <span>{notice.message}</span>
            </div>
          )}
          {/* === 头部卡片 === */}
          <div className="rounded-xl border bg-card p-5 card-shadow">
            <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-lg bg-primary/10 border border-primary/20 flex items-center justify-center">
                  <Server className="w-6 h-6 text-primary" />
                </div>
                <div>
                  <h2 className="font-display text-xl font-bold text-foreground">{subId}</h2>
                  <p className="text-xs text-muted-foreground flex items-center gap-1">
                    <span className="inline-block w-1.5 h-1.5 rounded-full bg-success"></span>
                    {d.overview.lastUsed}: {new Date().toLocaleString(locale)}
                  </p>
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button variant="outline" size="sm" className="text-pink-600 dark:text-pink-400 border-pink-200 dark:border-pink-500/30 hover:bg-pink-50 dark:hover:bg-pink-500/10 disabled:opacity-50" onClick={openTrafficResetDialog} disabled={resettingTraffic} title={resetPriceEnabled ? `¥${resetPriceNum.toFixed(2)}` : d.overview.trafficNotAvailable}>
                  <CreditCard className={`w-3.5 h-3.5 mr-1.5 ${resettingTraffic ? "animate-pulse" : ""}`} /> {resettingTraffic ? d.overview.resetTrafficCreating : resetPriceEnabled ? d.overview.resetTrafficPrice.replace("{price}", resetPriceNum.toFixed(2)) : d.overview.resetTraffic}
                </Button>
                <Button variant="outline" size="sm" className="text-primary border-primary/30 hover:bg-primary/10" onClick={() => setUUIDDialogOpen(true)} disabled={resettingUUID}>
                  <Shield className="w-3.5 h-3.5 mr-1.5" /> {resettingUUID ? d.overview.resettingUUID : d.overview.resetUUID}
                </Button>
                <Button variant="outline" size="sm" className="text-orange-600 dark:text-orange-400 border-orange-200 dark:border-orange-500/30 hover:bg-orange-50 dark:hover:bg-orange-500/10" onClick={handleResetToken}>
                  <RefreshCw className="w-3.5 h-3.5 mr-1.5" /> {d.overview.resetSubLink}
                </Button>
              </div>
            </div>
          </div>

          {/* === 产品资讯 + 使用统计 === */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="rounded-xl border bg-card p-5 card-shadow">
              <h3 className="text-sm font-medium text-muted-foreground mb-4">{d.overview.productInfo}</h3>
              <div className="grid grid-cols-3 gap-3">
                <div className="text-center p-3 rounded-lg bg-secondary/50">
                  <p className="text-xs text-muted-foreground mb-1">{d.overview.status}</p>
                  <p className={`font-bold ${user.status === "active" ? "text-success" : "text-destructive"}`}>
                    {user.status === "active" ? d.overview.active : d.overview.disabled}
                  </p>
                </div>
                <div className="text-center p-3 rounded-lg bg-secondary/50">
                  <p className="text-xs text-muted-foreground mb-1">{d.overview.expiry}</p>
                  <p className="font-bold text-foreground text-sm tabular-nums">{expireDate}</p>
                </div>
                <div className="text-center p-3 rounded-lg bg-secondary/50">
                  <p className="text-xs text-muted-foreground mb-1">{d.overview.nodeId}</p>
                  <p className="font-bold text-foreground tabular-nums">{subId.replace("LINK-", "")}</p>
                </div>
              </div>
            </div>

            <div className="rounded-xl border bg-card p-5 card-shadow">
              <h3 className="text-sm font-medium text-muted-foreground mb-4">{d.overview.monthlyUsage}</h3>
              <div className="grid grid-cols-2 gap-4">
                <div className="text-center">
                  <p className="font-display text-3xl font-bold text-primary tabular-nums">{totalFormatted.value}<span className="text-base font-normal text-muted-foreground ml-0.5">{totalFormatted.unit}</span></p>
                  <p className="text-xs text-muted-foreground mt-1">{d.overview.total}</p>
                </div>
                <div className="text-center">
                  <p className="font-display text-3xl font-bold text-foreground tabular-nums">{usedFormatted.value}<span className="text-base font-normal text-muted-foreground ml-0.5">{usedFormatted.unit}</span></p>
                  <p className="text-xs text-muted-foreground mt-1 flex items-center justify-center gap-1">{d.overview.usedTraffic} <RefreshCw className="w-3 h-3" /></p>
                </div>
              </div>
            </div>
          </div>

          {/* === 流量策略 === */}
          <div className="rounded-xl border bg-card p-5 card-shadow">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-foreground">{d.overview.trafficPolicy}</h3>
              <span className="text-xs text-muted-foreground">{d.overview.autoReset}</span>
            </div>
            <div className="relative w-full h-6 rounded-full bg-primary/15 overflow-hidden">
              <div
                className="h-full rounded-full bg-gradient-to-r from-primary/80 to-primary transition-all duration-500 flex items-center"
                style={{ width: `${Math.max(usedPct, 2)}%` }}
              >
                <span className="text-[10px] text-primary-foreground font-medium pl-3 whitespace-nowrap tabular-nums">
                  {d.overview.usedPercent.replace("{pct}", usedPct.toFixed(2))}
                </span>
              </div>
              {usedPct > 80 && (
                <div className="absolute right-0 top-0 h-full bg-destructive/50" style={{ width: `${100 - 80}%` }} />
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-2">{d.overview.trafficExceeded}</p>
          </div>

          {plans.length > 0 && (
            <div className="rounded-xl border bg-card p-5 space-y-4 card-shadow">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <h3 className="text-sm font-medium text-foreground">{d.overview.upgradeSubscription}</h3>
                  <p className="text-xs text-muted-foreground mt-1">{d.overview.upgradeDesc}</p>
                </div>
              </div>
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                {plans.map((plan) => (
                  <div key={plan.id} className="rounded-lg border p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="font-semibold">{plan.name}</div>
                        <div className="text-xs text-muted-foreground mt-1">
                          {bytesToGB(plan.traffic_limit)} GB {d.overview.perMonth}，{plan.device_limit || 3} {d.overview.devices}
                        </div>
                      </div>
                      {user.plan_id === plan.id && (
                        <span className="rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">{d.overview.currentPlan}</span>
                      )}
                    </div>
                    <div className="mt-4 grid grid-cols-3 gap-2">
                      {billingPeriods.map((item) => {
                        const buttonKey = `${plan.id}:${item.value}`
                        return (
                          <Button
                            key={item.value}
                            variant={item.value === "yearly" ? "default" : "outline"}
                            className="h-auto flex-col items-start gap-1 px-3 py-2"
                            disabled={buyingPlanKey !== null}
                            onClick={() => handleBuyPlan(plan, item.value)}
                          >
                            <span className="text-xs">{item.label}</span>
                            <span className="text-sm font-semibold tabular-nums">¥{periodPrice(plan, item.value)}</span>
                            {buyingPlanKey === buttonKey && <span className="text-[10px] opacity-80">{d.overview.creating}</span>}
                          </Button>
                        )
                      })}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* === 节点配置 === */}
          <div className="rounded-xl border bg-card p-5 space-y-5 card-shadow">
            <h3 className="text-sm font-medium text-foreground flex items-center gap-2">
              <span className="inline-block w-2 h-2 rounded-full bg-success"></span>
              {d.overview.nodeConfig}
            </h3>

            <div>
              <p className="text-xs text-muted-foreground mb-2">{d.overview.selectClientType}</p>
              <div className="relative">
                <select
                  value={clientType}
                  onChange={(e) => setClientType(e.target.value)}
                  className="w-full rounded-lg border bg-card px-4 py-2.5 text-sm appearance-none cursor-pointer pr-10 focus:outline-none focus:ring-2 focus:ring-primary/20"
                >
                  {clientOptions.map((item) => (
                    <option key={item.value} value={item.value}>{item.label}</option>
                  ))}
                </select>
                <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
              </div>
            </div>

            <div>
              <div className="flex items-center gap-2 mb-2">
                <span className="text-sm font-medium">{d.overview.mainSubLink}</span>
                <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded-full font-medium">{d.overview.recommended}</span>
                <button
                  onClick={() => handleCopy(mainLink, "main")}
                  className="ml-auto text-sm text-primary hover:text-primary/80 flex items-center gap-1"
                >
                  <Copy className="w-3.5 h-3.5" /> {copied === "main" ? d.common.copied : d.common.copy}
                </button>
              </div>
              <div
                className="rounded-lg bg-secondary/50 border px-4 py-3 cursor-pointer hover:bg-secondary/70 transition"
                onClick={() => handleCopy(mainLink, "main")}
              >
                <p className="text-xs text-foreground break-all font-mono">{mainLink}</p>
                <p className="text-[11px] text-muted-foreground mt-1">{d.overview.clickToCopyMain}</p>
              </div>
            </div>

            <div>
              <div className="flex items-center gap-2 mb-2">
                <span className="text-sm font-medium">{d.overview.backupSubLink}</span>
                <span className="text-xs bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-400 px-2 py-0.5 rounded-full font-medium">{d.overview.backupHint}</span>
                {backupLink && (
                  <button
                    onClick={() => handleCopy(backupLink, "backup")}
                    className="ml-auto text-sm text-primary hover:text-primary/80 flex items-center gap-1"
                  >
                    <Copy className="w-3.5 h-3.5" /> {copied === "backup" ? d.common.copied : d.common.copy}
                  </button>
                )}
              </div>
              {backupLink ? (
                <div
                  className="rounded-lg bg-secondary/50 border px-4 py-3 cursor-pointer hover:bg-secondary/70 transition"
                  onClick={() => handleCopy(backupLink, "backup")}
                >
                  <p className="text-xs text-foreground break-all font-mono">{backupLink}</p>
                  <p className="text-[11px] text-muted-foreground mt-1">{d.overview.clickToCopyBackup}</p>
                </div>
              ) : (
                <div className="rounded-lg bg-secondary/30 border border-dashed px-4 py-3">
                  <p className="text-[11px] text-muted-foreground">{d.overview.noBackupDomain}</p>
                </div>
              )}
            </div>

            <QRCodeDialog url={mainLink} title={d.overview.subQRTitle} />

            <div className="rounded-lg bg-secondary/30 border p-4">
              <h4 className="text-sm font-medium flex items-center gap-1.5 mb-3">
                <Info className="w-4 h-4 text-muted-foreground" /> {d.overview.subInfo}
              </h4>
              <div className="space-y-2 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">{d.overview.userId}</span>
                  <span className="font-mono text-xs">{subId}</span>
                </div>
              </div>
            </div>
          </div>
      </div>
      <AlertDialog open={trafficDialogOpen} onOpenChange={setTrafficDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{d.overview.trafficResetTitle}</AlertDialogTitle>
            <AlertDialogDescription>
              {d.overview.trafficResetDesc.replace("{price}", resetPriceNum.toFixed(2))}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="rounded-lg border bg-muted/40 p-4 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{d.overview.orderType}</span>
              <span className="font-medium">{d.overview.trafficResetOrder}</span>
            </div>
            <div className="mt-2 flex items-center justify-between">
              <span className="text-muted-foreground">{d.overview.amountDue}</span>
              <span className="font-semibold text-primary">¥{resetPriceNum.toFixed(2)}</span>
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={resettingTraffic}>{d.common.cancel}</AlertDialogCancel>
            <AlertDialogAction
              disabled={resettingTraffic}
              onClick={(event) => {
                event.preventDefault()
                void handleResetTrafficPayment()
              }}
            >
              {resettingTraffic ? d.overview.creatingPayment : d.overview.goToPay}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={uuidDialogOpen} onOpenChange={setUUIDDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{d.overview.uuidResetTitle}</AlertDialogTitle>
            <AlertDialogDescription>
              {d.overview.uuidResetDesc}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={resettingUUID}>{d.common.cancel}</AlertDialogCancel>
            <AlertDialogAction
              disabled={resettingUUID}
              onClick={(event) => {
                event.preventDefault()
                void handleResetUUID()
              }}
            >
              {resettingUUID ? d.overview.resettingUUID : d.overview.confirmReset}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
