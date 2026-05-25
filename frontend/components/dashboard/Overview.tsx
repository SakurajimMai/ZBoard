"use client"

import { useEffect, useState } from "react"
import {
  Copy, RefreshCw, Server, Shield,
  ChevronDown, Info
} from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  getMe, getSubscription, resetSubscriptionToken,
  getPublicSettings,
  resetMyTraffic, resetMyUUID,
  buildSubscriptionUrl,
  buildSubscriptionUrlFromBase,
} from "@/lib/api"
import QRCodeDialog from "@/components/dashboard/QRCodeDialog"

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

export default function Overview() {
  const [user, setUser] = useState<any>(null)
  const [subToken, setSubToken] = useState("")
  const [settings, setSettings] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [copied, setCopied] = useState<string | null>(null)
  const [clientType, setClientType] = useState("general")
  const [resettingTraffic, setResettingTraffic] = useState(false)
  const [resettingUUID, setResettingUUID] = useState(false)

  useEffect(() => {
    Promise.all([
      getMe(),
      getSubscription(),
      getPublicSettings().catch(() => ({ settings: {} })),
    ])
      .then(([meRes, subRes, settingsRes]) => {
        setUser(meRes.user)
        setSubToken(subRes.token)
        setSettings(settingsRes.settings)
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>
  if (!user) return <div className="text-red-500 p-8">无法加载用户信息</div>

  const target = clientType === "clash" ? "clash" : clientType === "singbox" ? "sing-box" : undefined
  const mainLink = buildSubscriptionUrl(subToken, target, settings)

  const backupDomain = settings.backup_subscription_domain || ""
  const backupLink = backupDomain ? buildSubscriptionUrlFromBase(backupDomain, subToken, target) : ""

  const totalBytes = user.traffic_limit || 0
  const usedBytes = user.traffic_used || 0
  const totalFormatted = formatBytes(totalBytes)
  const usedFormatted = formatBytes(usedBytes)
  const usedPct = totalBytes > 0 ? Math.min((usedBytes / totalBytes) * 100, 100) : 0

  const subId = generateSubId(subToken)
  const expireDate = user.expired_at ? new Date(user.expired_at).toLocaleDateString("zh-CN") : "无"

  const handleCopy = (text: string, id: string) => {
    navigator.clipboard.writeText(text)
    setCopied(id)
    setTimeout(() => setCopied(null), 2000)
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
  const balanceNum = Number((user as any)?.balance ?? 0)

  const handleResetTraffic = async () => {
    if (resettingTraffic) return
    if (!resetPriceEnabled) {
      alert("当前套餐未开放流量重置，请联系客服或升级套餐。")
      return
    }
    if (balanceNum < resetPriceNum) {
      alert(`余额不足：本次重置需要 ¥${resetPriceNum.toFixed(2)}，当前余额 ¥${balanceNum.toFixed(2)}，请先充值。`)
      return
    }
    if (!confirm(`本次重置流量将从余额扣费 ¥${resetPriceNum.toFixed(2)}，确认继续？`)) return
    setResettingTraffic(true)
    try {
      await resetMyTraffic()
      const meRes = await getMe()
      setUser(meRes.user)
    } catch (e: any) {
      alert(e?.message || "重置流量失败")
    } finally {
      setResettingTraffic(false)
    }
  }

  const handleResetUUID = async () => {
    if (resettingUUID) return
    if (!confirm("重置 UUID 后旧客户端将失效，需要重新导入订阅。确认继续？")) return
    setResettingUUID(true)
    try {
      await resetMyUUID()
      alert("UUID 已重置，节点同步任务已下发，请重新导入订阅。")
    } catch (e: any) {
      alert(e?.message || "重置 UUID 失败")
    } finally {
      setResettingUUID(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="space-y-6">
          {/* === 头部卡片 === */}
          <div className="rounded-xl border bg-card p-5">
            <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-lg bg-blue-50 border border-blue-100 flex items-center justify-center">
                  <Server className="w-6 h-6 text-blue-500" />
                </div>
                <div>
                  <h2 className="text-xl font-bold text-foreground">{subId}</h2>
                  <p className="text-xs text-muted-foreground flex items-center gap-1">
                    <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-500"></span>
                    最后使用: {new Date().toLocaleString("zh-CN")}
                  </p>
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button variant="outline" size="sm" className="text-pink-600 border-pink-200 hover:bg-pink-50 disabled:opacity-50" onClick={handleResetTraffic} disabled={resettingTraffic || !resetPriceEnabled} title={resetPriceEnabled ? `本次重置 ¥${resetPriceNum.toFixed(2)}` : "当前套餐未开放流量重置"}>
                  <RefreshCw className={`w-3.5 h-3.5 mr-1.5 ${resettingTraffic ? "animate-spin" : ""}`} /> {resettingTraffic ? "重置中..." : resetPriceEnabled ? `重置流量 ¥${resetPriceNum.toFixed(2)}` : "重置流量"}
                </Button>
                <Button variant="outline" size="sm" className="text-blue-600 border-blue-200 hover:bg-blue-50" onClick={handleResetUUID} disabled={resettingUUID}>
                  <Shield className="w-3.5 h-3.5 mr-1.5" /> {resettingUUID ? "重置中..." : "重置 UUID"}
                </Button>
                <Button variant="outline" size="sm" className="text-orange-600 border-orange-200 hover:bg-orange-50" onClick={handleResetToken}>
                  <RefreshCw className="w-3.5 h-3.5 mr-1.5" /> 重置节点订阅链接
                </Button>
              </div>
            </div>
          </div>

          {/* === 产品资讯 + 使用统计 === */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* 产品资讯 */}
            <div className="rounded-xl border bg-card p-5">
              <h3 className="text-sm font-medium text-muted-foreground mb-4">产品资讯</h3>
              <div className="grid grid-cols-3 gap-3">
                <div className="text-center p-3 rounded-lg bg-secondary/50">
                  <p className="text-xs text-muted-foreground mb-1">状态</p>
                  <p className={`font-bold ${user.status === "active" ? "text-green-600" : "text-red-500"}`}>
                    {user.status === "active" ? "可用" : "禁用"}
                  </p>
                </div>
                <div className="text-center p-3 rounded-lg bg-secondary/50">
                  <p className="text-xs text-muted-foreground mb-1">到期日</p>
                  <p className="font-bold text-foreground text-sm">{expireDate}</p>
                </div>
                <div className="text-center p-3 rounded-lg bg-secondary/50">
                  <p className="text-xs text-muted-foreground mb-1">节点标识</p>
                  <p className="font-bold text-foreground">{subId.replace("LINK-", "")}</p>
                </div>
              </div>
            </div>

            {/* 本月使用统计 */}
            <div className="rounded-xl border bg-card p-5">
              <h3 className="text-sm font-medium text-muted-foreground mb-4">本月使用统计</h3>
              <div className="grid grid-cols-2 gap-4">
                <div className="text-center">
                  <p className="text-3xl font-bold text-blue-600">{totalFormatted.value}<span className="text-base font-normal text-muted-foreground ml-0.5">{totalFormatted.unit}</span></p>
                  <p className="text-xs text-muted-foreground mt-1">全部</p>
                </div>
                <div className="text-center">
                  <p className="text-3xl font-bold text-foreground">{usedFormatted.value}<span className="text-base font-normal text-muted-foreground ml-0.5">{usedFormatted.unit}</span></p>
                  <p className="text-xs text-muted-foreground mt-1 flex items-center justify-center gap-1">已用流量 <RefreshCw className="w-3 h-3" /></p>
                </div>
              </div>
            </div>
          </div>

          {/* === 流量策略 === */}
          <div className="rounded-xl border bg-card p-5">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-foreground">流量策略</h3>
              <span className="text-xs text-muted-foreground">每月自动重置已用流量 (UTC)</span>
            </div>
            <div className="relative w-full h-6 rounded-full bg-blue-100 overflow-hidden">
              <div
                className="h-full rounded-full bg-gradient-to-r from-blue-400 to-blue-500 transition-all duration-500 flex items-center"
                style={{ width: `${Math.max(usedPct, 2)}%` }}
              >
                <span className="text-[10px] text-white font-medium pl-3 whitespace-nowrap">
                  本月使用统计 已使用 {usedPct.toFixed(2)}%
                </span>
              </div>
              {usedPct > 80 && (
                <div className="absolute right-0 top-0 h-full bg-red-400/60" style={{ width: `${100 - 80}%` }} />
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-2">若流量超过方案限制，将自动停用。</p>
          </div>

          {/* === 节点配置 === */}
          <div className="rounded-xl border bg-card p-5 space-y-5">
            <h3 className="text-sm font-medium text-foreground flex items-center gap-2">
              <span className="inline-block w-2 h-2 rounded-full bg-green-500"></span>
              节点配置
            </h3>

            {/* 客户端类型选择 */}
            <div>
              <p className="text-xs text-muted-foreground mb-2">选择工具类型</p>
              <div className="relative">
                <select
                  value={clientType}
                  onChange={(e) => setClientType(e.target.value)}
                  className="w-full rounded-lg border bg-card px-4 py-2.5 text-sm appearance-none cursor-pointer pr-10 focus:outline-none focus:ring-2 focus:ring-primary/20"
                >
                  <option value="general">[通用] 通用节点配置 - 多端通用</option>
                  <option value="clash">[Meta] Meta 节点配置 - 适用 Meta 兼容客户端</option>
                  <option value="singbox">[Box] Box 节点配置 - 适用 Box 兼容客户端</option>
                </select>
                <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
              </div>
            </div>

            {/* 主配置连结 */}
            <div>
              <div className="flex items-center gap-2 mb-2">
                <span className="text-sm font-medium">主节点订阅链接</span>
                <span className="text-xs bg-blue-100 text-blue-700 px-2 py-0.5 rounded-full font-medium">推荐</span>
                <button
                  onClick={() => handleCopy(mainLink, "main")}
                  className="ml-auto text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
                >
                  <Copy className="w-3.5 h-3.5" /> {copied === "main" ? "已复制" : "复制"}
                </button>
              </div>
              <div
                className="rounded-lg bg-secondary/50 border px-4 py-3 cursor-pointer hover:bg-secondary/70 transition"
                onClick={() => handleCopy(mainLink, "main")}
              >
                <p className="text-xs text-foreground break-all font-mono">{mainLink}</p>
                <p className="text-[11px] text-muted-foreground mt-1">点击复制主节点订阅链接</p>
              </div>
            </div>

            {/* 备用节点订阅链接 */}
            <div>
              <div className="flex items-center gap-2 mb-2">
                <span className="text-sm font-medium">备用节点订阅链接</span>
                <span className="text-xs bg-amber-100 text-amber-700 px-2 py-0.5 rounded-full font-medium">主链接不可用时使用</span>
                {backupLink && (
                  <button
                    onClick={() => handleCopy(backupLink, "backup")}
                    className="ml-auto text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
                  >
                    <Copy className="w-3.5 h-3.5" /> {copied === "backup" ? "已复制" : "复制"}
                  </button>
                )}
              </div>
              {backupLink ? (
                <div
                  className="rounded-lg bg-secondary/50 border px-4 py-3 cursor-pointer hover:bg-secondary/70 transition"
                  onClick={() => handleCopy(backupLink, "backup")}
                >
                  <p className="text-xs text-foreground break-all font-mono">{backupLink}</p>
                  <p className="text-[11px] text-muted-foreground mt-1">点击复制备用节点订阅链接</p>
                </div>
              ) : (
                <div className="rounded-lg bg-secondary/30 border border-dashed px-4 py-3">
                  <p className="text-[11px] text-muted-foreground">站点尚未配置备用域名，请联系管理员开启。</p>
                </div>
              )}
            </div>

            {/* 生成二维码 */}
            <QRCodeDialog url={mainLink} title="节点订阅链接二维码" />

            {/* 节点订阅信息 */}
            <div className="rounded-lg bg-secondary/30 border p-4">
              <h4 className="text-sm font-medium flex items-center gap-1.5 mb-3">
                <Info className="w-4 h-4 text-muted-foreground" /> 节点订阅信息
              </h4>
              <div className="space-y-2 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">用户标识</span>
                  <span className="font-mono text-xs">{subId}</span>
                </div>
              </div>
            </div>
          </div>
      </div>
    </div>
  )
}
