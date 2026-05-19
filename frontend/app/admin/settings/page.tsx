"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { 
  Shield, 
  Mail, 
  Bell, 
  Server, 
  Globe, 
  Settings, 
  Eye, 
  EyeOff,
  Copy,
  RefreshCw,
  Info
} from "lucide-react"

function Toggle({ checked, onChange }: { checked: boolean; onChange?: (val: boolean) => void }) {
  return (
    <button
      type="button"
      onClick={() => onChange?.(!checked)}
      className={`w-11 h-6 rounded-full relative transition-colors ${
        checked ? "bg-primary" : "bg-secondary border border-border"
      }`}
    >
      <div
        className={`w-5 h-5 rounded-full bg-white shadow-sm absolute top-0.5 transition-all ${
          checked ? "right-0.5" : "left-0.5"
        }`}
      />
    </button>
  )
}

function SectionCard({ 
  icon: Icon, 
  title, 
  description, 
  children 
}: { 
  icon: React.ElementType
  title: string
  description: string
  children: React.ReactNode 
}) {
  return (
    <div className="rounded-2xl border border-border bg-card overflow-hidden">
      <div className="px-6 py-4 border-b border-border bg-secondary/30 flex items-center gap-3">
        <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
          <Icon className="w-5 h-5 text-primary" />
        </div>
        <div>
          <h2 className="font-semibold text-foreground">{title}</h2>
          <p className="text-xs text-muted-foreground">{description}</p>
        </div>
      </div>
      <div className="p-6">{children}</div>
    </div>
  )
}

export default function AdminSettingsPage() {
  const [showApiKey, setShowApiKey] = useState(false)
  const [activeTab, setActiveTab] = useState("security")
  const [captchaProvider, setCaptchaProvider] = useState<"none" | "turnstile" | "recaptcha" | "hcaptcha">("turnstile")

  const tabs = [
    { id: "security", label: "安全设置", icon: Shield },
    { id: "subscription", label: "订阅设置", icon: Bell },
    { id: "node", label: "节点通讯", icon: Server },
    { id: "email", label: "邮件配置", icon: Mail },
    { id: "seo", label: "站点SEO", icon: Globe },
    { id: "basic", label: "基础设置", icon: Settings },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">系统设置</h1>
        <p className="text-sm text-muted-foreground mt-1">配置平台安全、订阅、节点通讯、邮件及SEO等核心参数。</p>
      </div>

      {/* Tabs */}
      <div className="flex flex-wrap gap-2 p-1 bg-secondary/50 rounded-xl border border-border">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === tab.id
                ? "bg-card text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground hover:bg-card/50"
            }`}
          >
            <tab.icon className="w-4 h-4" />
            <span className="hidden sm:inline">{tab.label}</span>
          </button>
        ))}
      </div>

      {/* Security Settings */}
      {activeTab === "security" && (
        <div className="space-y-6">
          <SectionCard
            icon={Shield}
            title="安全设置"
            description="配置邮箱验证、防机器人等安全策略"
          >
            <div className="space-y-6">
              {/* Toggle Options */}
              <div className="space-y-4">
                {[
                  { label: "强制邮箱验证", desc: "用户注册时必须验证邮箱", checked: true },
                  { label: "登录邮箱验证码", desc: "登录时发送验证码到邮箱", checked: false },
                  { label: "启用图形验证码", desc: "注册/登录页面显示图形验证码", checked: true },
                  { label: "登录失败锁定", desc: "连续5次失败后锁定账户30分钟", checked: true },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between py-2">
                    <div>
                      <span className="text-sm font-medium text-foreground">{item.label}</span>
                      <p className="text-xs text-muted-foreground mt-0.5">{item.desc}</p>
                    </div>
                    <Toggle checked={item.checked} />
                  </div>
                ))}
              </div>

              {/* Bot Protection — single provider selection */}
              <div className="border-t border-border pt-6 space-y-4">
                <div>
                  <h3 className="text-sm font-medium text-foreground mb-1">防机器人验证</h3>
                  <p className="text-xs text-muted-foreground">每次只能启用一种验证服务，选择后填写对应密钥即可。</p>
                </div>

                {/* Provider Selector */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                  {[
                    { value: "none",      label: "不启用",   desc: "关闭人机验证" },
                    { value: "turnstile", label: "Turnstile", desc: "Cloudflare · 推荐" },
                    { value: "recaptcha", label: "reCAPTCHA", desc: "Google · v2/v3" },
                    { value: "hcaptcha",  label: "hCaptcha",  desc: "隐私友好" },
                  ].map((opt) => (
                    <button
                      key={opt.value}
                      type="button"
                      onClick={() => setCaptchaProvider(opt.value as typeof captchaProvider)}
                      className={`flex flex-col items-start gap-0.5 p-4 rounded-xl border text-left transition-all ${
                        captchaProvider === opt.value
                          ? "border-primary bg-primary/5 shadow-sm"
                          : "border-border bg-secondary/30 hover:border-primary/40"
                      }`}
                    >
                      <span className="text-sm font-medium text-foreground">{opt.label}</span>
                      <span className="text-xs text-muted-foreground">{opt.desc}</span>
                    </button>
                  ))}
                </div>

                {/* Turnstile Config */}
                {captchaProvider === "turnstile" && (
                  <div className="rounded-xl border border-border bg-secondary/20 p-5 space-y-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-semibold text-foreground">Cloudflare Turnstile</span>
                        <a
                          href="https://dash.cloudflare.com/?to=/:account/turnstile"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-xs text-primary hover:underline"
                        >
                          获取密钥 →
                        </a>
                      </div>
                    </div>
                    <div className="flex items-start gap-2">
                      <Info className="w-4 h-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                      <p className="text-xs text-muted-foreground">免费、隐私友好，无需用户主动操作即可完成验证，不收集个人数据。</p>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wide">Site Key</label>
                        <Input placeholder="0x4AAAAAAA..." className="bg-card border-border font-mono text-sm" />
                      </div>
                      <div>
                        <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wide">Secret Key</label>
                        <Input type="password" placeholder="0x4AAAAAAA..." className="bg-card border-border font-mono text-sm" />
                      </div>
                    </div>
                    <div>
                      <label className="text-xs font-medium text-muted-foreground mb-2 block uppercase tracking-wide">验证模式</label>
                      <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                        {[
                          { value: "managed",         label: "托管模式",  desc: "自动选择最佳方式" },
                          { value: "non-interactive", label: "非交互式",  desc: "完全无感验证" },
                          { value: "invisible",       label: "隐式验证",  desc: "可疑时才显示" },
                        ].map((opt) => (
                          <label key={opt.value} className="flex items-start gap-2.5 p-3 rounded-lg border border-border bg-card cursor-pointer hover:border-primary/50 transition-colors">
                            <input type="radio" name="turnstileMode" value={opt.value} defaultChecked={opt.value === "managed"} className="mt-0.5 w-4 h-4 accent-primary" />
                            <div>
                              <span className="text-sm font-medium text-foreground">{opt.label}</span>
                              <p className="text-xs text-muted-foreground">{opt.desc}</p>
                            </div>
                          </label>
                        ))}
                      </div>
                    </div>
                    <div className="border-t border-border pt-4 space-y-3">
                      <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">启用页面</p>
                      {[
                        { label: "注册页", checked: true },
                        { label: "登录页", checked: true },
                        { label: "找回密码", checked: true },
                        { label: "提交工单", checked: false },
                      ].map((item) => (
                        <div key={item.label} className="flex items-center justify-between">
                          <span className="text-sm text-foreground">{item.label}</span>
                          <Toggle checked={item.checked} />
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* reCAPTCHA Config */}
                {captchaProvider === "recaptcha" && (
                  <div className="rounded-xl border border-border bg-secondary/20 p-5 space-y-4">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-semibold text-foreground">Google reCAPTCHA</span>
                      <a
                        href="https://www.google.com/recaptcha/admin"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-xs text-primary hover:underline"
                      >
                        获取密钥 →
                      </a>
                    </div>
                    <div className="flex items-start gap-2">
                      <Info className="w-4 h-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                      <p className="text-xs text-muted-foreground">Google 提供的人机验证服务，v3 为无感评分模式，v2 显示勾选框或图片挑战。</p>
                    </div>
                    <div>
                      <label className="text-xs font-medium text-muted-foreground mb-2 block uppercase tracking-wide">版本</label>
                      <div className="flex gap-3">
                        {["reCAPTCHA v3 (推荐)", "reCAPTCHA v2 勾选框", "reCAPTCHA v2 隐形"].map((v) => (
                          <label key={v} className="flex items-center gap-2 px-3 py-2 rounded-lg border border-border bg-card cursor-pointer hover:border-primary/50 transition-colors text-sm">
                            <input type="radio" name="recaptchaVersion" defaultChecked={v.includes("v3")} className="w-4 h-4 accent-primary" />
                            {v}
                          </label>
                        ))}
                      </div>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wide">Site Key</label>
                        <Input placeholder="6LeIxAcT..." className="bg-card border-border font-mono text-sm" />
                      </div>
                      <div>
                        <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wide">Secret Key</label>
                        <Input type="password" placeholder="6LeIxAcT..." className="bg-card border-border font-mono text-sm" />
                      </div>
                      <div>
                        <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wide">v3 评分阈值</label>
                        <Input type="number" min="0" max="1" step="0.1" defaultValue="0.5" className="bg-card border-border" />
                        <p className="text-xs text-muted-foreground mt-1">低于此分值视为机器人 (0.0 ~ 1.0)</p>
                      </div>
                    </div>
                    <div className="border-t border-border pt-4 space-y-3">
                      <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">启用页面</p>
                      {[
                        { label: "注册页", checked: true },
                        { label: "登录页", checked: true },
                        { label: "找回密码", checked: true },
                        { label: "提交工单", checked: false },
                      ].map((item) => (
                        <div key={item.label} className="flex items-center justify-between">
                          <span className="text-sm text-foreground">{item.label}</span>
                          <Toggle checked={item.checked} />
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* hCaptcha Config */}
                {captchaProvider === "hcaptcha" && (
                  <div className="rounded-xl border border-border bg-secondary/20 p-5 space-y-4">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-semibold text-foreground">hCaptcha</span>
                      <a
                        href="https://dashboard.hcaptcha.com/signup"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-xs text-primary hover:underline"
                      >
                        获取密钥 →
                      </a>
                    </div>
                    <div className="flex items-start gap-2">
                      <Info className="w-4 h-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                      <p className="text-xs text-muted-foreground">注重隐私保护的人机验证服务，不依赖 Google 基础设施，支持在中国大陆访问。</p>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wide">Site Key</label>
                        <Input placeholder="10000000-ffff-ffff..." className="bg-card border-border font-mono text-sm" />
                      </div>
                      <div>
                        <label className="text-xs font-medium text-muted-foreground mb-1.5 block uppercase tracking-wide">Secret Key</label>
                        <Input type="password" placeholder="0x0000000000000000..." className="bg-card border-border font-mono text-sm" />
                      </div>
                    </div>
                    <div>
                      <label className="text-xs font-medium text-muted-foreground mb-2 block uppercase tracking-wide">难度等级</label>
                      <div className="flex gap-3">
                        {["自动 (推荐)", "简单", "困难"].map((lvl) => (
                          <label key={lvl} className="flex items-center gap-2 px-3 py-2 rounded-lg border border-border bg-card cursor-pointer hover:border-primary/50 transition-colors text-sm">
                            <input type="radio" name="hcaptchaLevel" defaultChecked={lvl.includes("自动")} className="w-4 h-4 accent-primary" />
                            {lvl}
                          </label>
                        ))}
                      </div>
                    </div>
                    <div className="border-t border-border pt-4 space-y-3">
                      <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">启用页面</p>
                      {[
                        { label: "注册页", checked: true },
                        { label: "登录页", checked: true },
                        { label: "找回密码", checked: true },
                        { label: "提交工单", checked: false },
                      ].map((item) => (
                        <div key={item.label} className="flex items-center justify-between">
                          <span className="text-sm text-foreground">{item.label}</span>
                          <Toggle checked={item.checked} />
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {captchaProvider === "none" && (
                  <div className="rounded-xl border border-border bg-secondary/20 p-5 flex items-start gap-3">
                    <Info className="w-4 h-4 text-amber-500 mt-0.5 flex-shrink-0" />
                    <p className="text-sm text-muted-foreground">当前未启用任何人机验证，注册和登录接口将暴露于自动化攻击风险中。建议至少开启一种验证方式。</p>
                  </div>
                )}
              </div>

              <div className="border-t border-border pt-6 space-y-4">
                <h3 className="text-sm font-medium text-foreground">邮箱后缀白名单</h3>
                <div className="flex items-start gap-2">
                  <Info className="w-4 h-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                  <p className="text-xs text-muted-foreground">每行一个邮箱后缀，留空则不限制。例如：gmail.com、outlook.com</p>
                </div>
                <textarea
                  className="w-full h-32 px-4 py-3 rounded-xl bg-secondary border border-border text-sm resize-none focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
                  placeholder="gmail.com&#10;outlook.com&#10;qq.com&#10;163.com"
                  defaultValue="gmail.com&#10;outlook.com&#10;qq.com&#10;163.com&#10;hotmail.com"
                />
              </div>

              <div className="border-t border-border pt-6 space-y-4">
                <h3 className="text-sm font-medium text-foreground">后台管理路径</h3>
                <div className="flex items-start gap-2">
                  <Info className="w-4 h-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                  <p className="text-xs text-muted-foreground">自定义后台管理入口路径，提高安全性。修改后需重新登录。</p>
                </div>
                <div className="flex gap-3">
                  <div className="flex-1 flex items-center bg-secondary rounded-xl border border-border overflow-hidden">
                    <span className="px-4 text-sm text-muted-foreground border-r border-border">https://zboard.io/</span>
                    <Input 
                      defaultValue="admin-secure-xyz" 
                      className="border-0 bg-transparent focus-visible:ring-0" 
                    />
                  </div>
                  <Button variant="outline" size="icon" className="shrink-0">
                    <RefreshCw className="w-4 h-4" />
                  </Button>
                </div>
              </div>

              <div className="flex justify-end pt-4">
                <Button className="bg-primary text-primary-foreground hover:bg-primary/90">保存安全设置</Button>
              </div>
            </div>
          </SectionCard>
        </div>
      )}

      {/* Subscription Settings */}
      {activeTab === "subscription" && (
        <div className="space-y-6">
          <SectionCard
            icon={Bell}
            title="订阅提醒设置"
            description="配置用户订阅到期、流量告急等提醒规则"
          >
            <div className="space-y-6">
              <div className="space-y-4">
                {[
                  { label: "启用到期提醒", desc: "订阅即将到期时发送邮件提醒", checked: true },
                  { label: "启用流量告急提醒", desc: "流量使用达到阈值时发送提醒", checked: true },
                  { label: "启用续费成功通知", desc: "用户成功续费后发送确认邮件", checked: true },
                  { label: "启用新订单通知", desc: "有新订单时通知管理员", checked: true },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between py-2">
                    <div>
                      <span className="text-sm font-medium text-foreground">{item.label}</span>
                      <p className="text-xs text-muted-foreground mt-0.5">{item.desc}</p>
                    </div>
                    <Toggle checked={item.checked} />
                  </div>
                ))}
              </div>

              <div className="border-t border-border pt-6 grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">到期提醒提前天数</label>
                  <Input type="number" defaultValue="3" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">订阅到期前几天发送提醒</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">流量告急阈值 (%)</label>
                  <Input type="number" defaultValue="80" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">流量使用达到百分比时提醒</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">提醒发送间隔 (小时)</label>
                  <Input type="number" defaultValue="24" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">同一类型提醒的最小间隔</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">最大提醒次数</label>
                  <Input type="number" defaultValue="3" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">同一订阅周期内最多提醒次数</p>
                </div>
              </div>

              <div className="flex justify-end pt-4">
                <Button className="bg-primary text-primary-foreground hover:bg-primary/90">保存提醒设置</Button>
              </div>
            </div>
          </SectionCard>

          <SectionCard
            icon={RefreshCw}
            title="流量重置设置"
            description="配置用户流量的重置方式和规则"
          >
            <div className="space-y-6">
              <div>
                <label className="text-sm font-medium text-foreground mb-3 block">流量重置方式</label>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  {[
                    { value: "monthly", label: "每月固定日", desc: "每月1号重置" },
                    { value: "purchase", label: "购买日重置", desc: "按购买日期循环" },
                    { value: "never", label: "不自动重置", desc: "流量用完为止" },
                  ].map((option) => (
                    <label
                      key={option.value}
                      className="flex items-start gap-3 p-4 rounded-xl border border-border bg-secondary/30 cursor-pointer hover:border-primary/50 transition-colors"
                    >
                      <input
                        type="radio"
                        name="resetMode"
                        value={option.value}
                        defaultChecked={option.value === "purchase"}
                        className="mt-1 w-4 h-4 text-primary"
                      />
                      <div>
                        <span className="text-sm font-medium text-foreground">{option.label}</span>
                        <p className="text-xs text-muted-foreground mt-0.5">{option.desc}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">固定重置日期</label>
                  <Input type="number" min="1" max="28" defaultValue="1" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">选择每月固定日时生效 (1-28)</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">重置时间</label>
                  <Input type="time" defaultValue="00:00" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">流量重置的具体时间点</p>
                </div>
              </div>

              <div className="space-y-4">
                {[
                  { label: "允许流量结转", desc: "未使用的流量结转到下个周期", checked: false },
                  { label: "超量后限速", desc: "流量用完后降速而非断开", checked: true },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between py-2">
                    <div>
                      <span className="text-sm font-medium text-foreground">{item.label}</span>
                      <p className="text-xs text-muted-foreground mt-0.5">{item.desc}</p>
                    </div>
                    <Toggle checked={item.checked} />
                  </div>
                ))}
              </div>

              <div>
                <label className="text-sm font-medium text-foreground mb-2 block">超量限速速率 (Mbps)</label>
                <Input type="number" defaultValue="1" className="bg-secondary border-border w-full md:w-1/2" />
                <p className="text-xs text-muted-foreground mt-1">流量超额后的限制速度</p>
              </div>

              <div className="flex justify-end pt-4">
                <Button className="bg-primary text-primary-foreground hover:bg-primary/90">保存流量设置</Button>
              </div>
            </div>
          </SectionCard>
        </div>
      )}

      {/* Node Communication Settings */}
      {activeTab === "node" && (
        <div className="space-y-6">
          <SectionCard
            icon={Server}
            title="节点通讯设置"
            description="配置节点与面板之间的通讯参数"
          >
            <div className="space-y-6">
              <div>
                <label className="text-sm font-medium text-foreground mb-2 block">节点通讯密钥</label>
                <div className="flex items-start gap-2">
                  <Info className="w-4 h-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                  <p className="text-xs text-muted-foreground">此密钥用于节点与面板之间的安全通讯，请妥善保管。</p>
                </div>
                <div className="flex gap-2 mt-2">
                  <div className="flex-1 relative">
                    <Input
                      type={showApiKey ? "text" : "password"}
                      defaultValue="zb_sk_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
                      className="bg-secondary border-border pr-20 font-mono text-sm"
                      readOnly
                    />
                    <div className="absolute right-2 top-1/2 -translate-y-1/2 flex gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7"
                        onClick={() => setShowApiKey(!showApiKey)}
                      >
                        {showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </Button>
                      <Button variant="ghost" size="icon" className="h-7 w-7">
                        <Copy className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                  <Button variant="outline" className="gap-2">
                    <RefreshCw className="w-4 h-4" />
                    <span className="hidden sm:inline">重新生成</span>
                  </Button>
                </div>
              </div>

              <div className="border-t border-border pt-6">
                <label className="text-sm font-medium text-foreground mb-3 block">通讯协议</label>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  {[
                    { value: "grpc", label: "gRPC", desc: "高性能、低延迟，推荐" },
                    { value: "http", label: "HTTP API", desc: "兼容性好，配置简单" },
                  ].map((option) => (
                    <label
                      key={option.value}
                      className="flex items-start gap-3 p-4 rounded-xl border border-border bg-secondary/30 cursor-pointer hover:border-primary/50 transition-colors"
                    >
                      <input
                        type="radio"
                        name="protocol"
                        value={option.value}
                        defaultChecked={option.value === "grpc"}
                        className="mt-1 w-4 h-4 text-primary"
                      />
                      <div>
                        <span className="text-sm font-medium text-foreground">{option.label}</span>
                        <p className="text-xs text-muted-foreground mt-0.5">{option.desc}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              <div className="border-t border-border pt-6 grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">节点拉取数据间隔 (秒)</label>
                  <Input type="number" defaultValue="60" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">节点从面板获取用户配置的频率</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">节点推送数据间隔 (秒)</label>
                  <Input type="number" defaultValue="60" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">节点向面板报告流量数据的频率</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">节点心跳间隔 (秒)</label>
                  <Input type="number" defaultValue="30" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">节点存活检测的心跳间隔</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">离线判定阈值 (秒)</label>
                  <Input type="number" defaultValue="120" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">超过此时间无心跳判定为离线</p>
                </div>
              </div>

              <div className="border-t border-border pt-6 space-y-4">
                {[
                  { label: "启用 TLS 加密", desc: "节点与面板通讯使用 TLS 加密", checked: true },
                  { label: "启用 IP 白名单", desc: "仅允许白名单内的节点 IP 连接", checked: false },
                  { label: "启用流量统计", desc: "统计并记录节点流量数据", checked: true },
                  { label: "启用在线用户统计", desc: "实时统计各节点在线用户数", checked: true },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between py-2">
                    <div>
                      <span className="text-sm font-medium text-foreground">{item.label}</span>
                      <p className="text-xs text-muted-foreground mt-0.5">{item.desc}</p>
                    </div>
                    <Toggle checked={item.checked} />
                  </div>
                ))}
              </div>

              <div>
                <label className="text-sm font-medium text-foreground mb-2 block">节点 IP 白名单</label>
                <textarea
                  className="w-full h-24 px-4 py-3 rounded-xl bg-secondary border border-border text-sm resize-none focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary font-mono"
                  placeholder="每行一个 IP 地址&#10;192.168.1.1&#10;10.0.0.0/24"
                />
              </div>

              <div className="flex justify-end pt-4">
                <Button className="bg-primary text-primary-foreground hover:bg-primary/90">保存节点设置</Button>
              </div>
            </div>
          </SectionCard>
        </div>
      )}

      {/* Email SMTP Settings */}
      {activeTab === "email" && (
        <div className="space-y-6">
          <SectionCard
            icon={Mail}
            title="SMTP 邮件配置"
            description="配置邮件发送服务器参数"
          >
            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">SMTP 服务器</label>
                  <Input defaultValue="smtp.gmail.com" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">SMTP 端口</label>
                  <Input type="number" defaultValue="587" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">SMTP 用户名</label>
                  <Input defaultValue="noreply@zboard.io" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">SMTP 密码</label>
                  <Input type="password" placeholder="请输入 SMTP 密码" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">发件人名称</label>
                  <Input defaultValue="Zboard" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">发件人邮箱</label>
                  <Input defaultValue="noreply@zboard.io" className="bg-secondary border-border" />
                </div>
              </div>

              <div>
                <label className="text-sm font-medium text-foreground mb-3 block">加密方式</label>
                <div className="flex flex-wrap gap-3">
                  {["TLS", "SSL", "STARTTLS", "无加密"].map((enc) => (
                    <label
                      key={enc}
                      className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-secondary/30 cursor-pointer hover:border-primary/50 transition-colors"
                    >
                      <input
                        type="radio"
                        name="encryption"
                        value={enc}
                        defaultChecked={enc === "TLS"}
                        className="w-4 h-4 text-primary"
                      />
                      <span className="text-sm text-foreground">{enc}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div className="space-y-4">
                {[
                  { label: "启用 SMTP 认证", desc: "使用用户名密码进行 SMTP 认证", checked: true },
                  { label: "验证 SSL 证书", desc: "验证服务器 SSL 证书有效性", checked: true },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between py-2">
                    <div>
                      <span className="text-sm font-medium text-foreground">{item.label}</span>
                      <p className="text-xs text-muted-foreground mt-0.5">{item.desc}</p>
                    </div>
                    <Toggle checked={item.checked} />
                  </div>
                ))}
              </div>

              <div className="flex flex-col sm:flex-row justify-end gap-3 pt-4">
                <Button variant="outline" className="gap-2">
                  <Mail className="w-4 h-4" />
                  发送测试邮件
                </Button>
                <Button className="bg-primary text-primary-foreground hover:bg-primary/90">保存邮件配置</Button>
              </div>
            </div>
          </SectionCard>

          <SectionCard
            icon={Mail}
            title="邮件模板设置"
            description="自定义各类邮件通知的模板内容"
          >
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium text-foreground mb-2 block">选择模板</label>
                <select className="w-full md:w-1/2 px-4 py-2.5 rounded-xl bg-secondary border border-border text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary">
                  <option>注册验证码邮件</option>
                  <option>登录验证码邮件</option>
                  <option>密码重置邮件</option>
                  <option>订阅到期提醒</option>
                  <option>流量告急提醒</option>
                  <option>订单支付成功</option>
                  <option>工单回复通知</option>
                </select>
              </div>
              <div>
                <label className="text-sm font-medium text-foreground mb-2 block">邮件主题</label>
                <Input defaultValue="【Zboard】您的验证码是 {{code}}" className="bg-secondary border-border" />
              </div>
              <div>
                <label className="text-sm font-medium text-foreground mb-2 block">邮件内容 (支持 HTML)</label>
                <textarea
                  className="w-full h-48 px-4 py-3 rounded-xl bg-secondary border border-border text-sm resize-none focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary font-mono"
                  defaultValue={`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif;">
  <h2>验证码</h2>
  <p>您好，您的验证码是：</p>
  <p style="font-size: 24px; font-weight: bold; color: #6366f1;">{{code}}</p>
  <p>验证码有效期为 10 分钟。</p>
  <p>如非本人操作，请忽略此邮件。</p>
</body>
</html>`}
                />
              </div>
              <div className="flex items-start gap-2">
                <Info className="w-4 h-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                <p className="text-xs text-muted-foreground">
                  可用变量: {"{{code}}"} 验证码、{"{{username}}"} 用户名、{"{{email}}"} 邮箱、{"{{expire_time}}"} 过期时间、{"{{site_name}}"} 站点名称
                </p>
              </div>
              <div className="flex justify-end pt-2">
                <Button className="bg-primary text-primary-foreground hover:bg-primary/90">保存模板</Button>
              </div>
            </div>
          </SectionCard>
        </div>
      )}

      {/* SEO Settings */}
      {activeTab === "seo" && (
        <div className="space-y-6">
          <SectionCard
            icon={Globe}
            title="站点 SEO 配置"
            description="优化搜索引擎收录与展示效果"
          >
            <div className="space-y-6">
              <div className="grid grid-cols-1 gap-4">
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">网站标题 (Title)</label>
                  <Input defaultValue="Zboard - 高速稳定的商业订阅机场" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">建议 50-60 个字符</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">网站描述 (Description)</label>
                  <textarea
                    className="w-full h-20 px-4 py-3 rounded-xl bg-secondary border border-border text-sm resize-none focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
                    defaultValue="Zboard 提供高速、稳定、安全的全球网络访问服务。支持多种协议，覆盖数十个国家节点，7x24 小时技术支持。"
                  />
                  <p className="text-xs text-muted-foreground mt-1">建议 150-160 个字符</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">关键词 (Keywords)</label>
                  <Input 
                    defaultValue="机场, VPN, 代理, 科学上网, 翻墙, Shadowsocks, V2Ray, Trojan" 
                    className="bg-secondary border-border" 
                  />
                  <p className="text-xs text-muted-foreground mt-1">多个关键词用英文逗号分隔</p>
                </div>
              </div>

              <div className="border-t border-border pt-6 grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">OG 图片 URL</label>
                  <Input defaultValue="https://zboard.io/og-image.png" className="bg-secondary border-border" />
                  <p className="text-xs text-muted-foreground mt-1">社交媒体分享时显示的图片</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">Twitter Card 类型</label>
                  <select className="w-full px-4 py-2.5 rounded-xl bg-secondary border border-border text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary">
                    <option>summary_large_image</option>
                    <option>summary</option>
                  </select>
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">Favicon URL</label>
                  <Input defaultValue="/favicon.ico" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">规范链接 (Canonical URL)</label>
                  <Input defaultValue="https://zboard.io" className="bg-secondary border-border" />
                </div>
              </div>

              <div className="border-t border-border pt-6 space-y-4">
                {[
                  { label: "允许搜索引擎索引", desc: "在 robots.txt 中允许爬虫抓取", checked: true },
                  { label: "生成 Sitemap", desc: "自动生成并更新站点地图", checked: true },
                  { label: "启用结构化数据", desc: "添加 JSON-LD 结构化数据标记", checked: true },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between py-2">
                    <div>
                      <span className="text-sm font-medium text-foreground">{item.label}</span>
                      <p className="text-xs text-muted-foreground mt-0.5">{item.desc}</p>
                    </div>
                    <Toggle checked={item.checked} />
                  </div>
                ))}
              </div>

              <div className="border-t border-border pt-6">
                <label className="text-sm font-medium text-foreground mb-2 block">自定义 Head 代码</label>
                <textarea
                  className="w-full h-32 px-4 py-3 rounded-xl bg-secondary border border-border text-sm resize-none focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary font-mono"
                  placeholder="<!-- 在此添加自定义的 HTML head 代码 -->&#10;<!-- 例如: Google Analytics, Facebook Pixel 等 -->"
                  defaultValue={`<!-- Google Analytics -->
<script async src="https://www.googletagmanager.com/gtag/js?id=G-XXXXXXXXXX"></script>`}
                />
              </div>

              <div className="flex justify-end pt-4">
                <Button className="bg-primary text-primary-foreground hover:bg-primary/90">保存 SEO 配置</Button>
              </div>
            </div>
          </SectionCard>
        </div>
      )}

      {/* Basic Settings */}
      {activeTab === "basic" && (
        <div className="space-y-6">
          <SectionCard
            icon={Settings}
            title="平台基础信息"
            description="配置平台名称、域名等基础信息"
          >
            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">平台名称</label>
                  <Input defaultValue="Zboard" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">平台域名</label>
                  <Input defaultValue="https://zboard.io" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">订阅下发域名</label>
                  <Input defaultValue="https://sub.zboard.io" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">客服 Telegram</label>
                  <Input defaultValue="@zboard_support" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">客服邮箱</label>
                  <Input defaultValue="support@zboard.io" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">默认语言</label>
                  <select className="w-full px-4 py-2.5 rounded-xl bg-secondary border border-border text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary">
                    <option>简体中文</option>
                    <option>繁体中文</option>
                    <option>English</option>
                    <option>日本語</option>
                  </select>
                </div>
              </div>
              <div className="flex justify-end pt-4">
                <Button className="bg-primary text-primary-foreground hover:bg-primary/90">保存基础设置</Button>
              </div>
            </div>
          </SectionCard>

          <SectionCard
            icon={Settings}
            title="注册与用户设置"
            description="配置用户注册、试用等相关选项"
          >
            <div className="space-y-6">
              <div className="space-y-4">
                {[
                  { label: "允许新用户注册", desc: "关闭后仅能通过邀请码注册", checked: true },
                  { label: "邮箱验证码注册", desc: "注册时需要邮箱验证码验证", checked: true },
                  { label: "邀请码限制注册", desc: "必须填写有效邀请码才能注册", checked: false },
                  { label: "新用户赠送试用流量", desc: "新注册用户自动获得试用流量", checked: true },
                  { label: "启用邀请返利", desc: "邀请新用户可获得返利奖励", checked: true },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between py-2">
                    <div>
                      <span className="text-sm font-medium text-foreground">{item.label}</span>
                      <p className="text-xs text-muted-foreground mt-0.5">{item.desc}</p>
                    </div>
                    <Toggle checked={item.checked} />
                  </div>
                ))}
              </div>

              <div className="border-t border-border pt-6 grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">试用流量 (GB)</label>
                  <Input defaultValue="2" type="number" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">试用有效期 (天)</label>
                  <Input defaultValue="3" type="number" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">邀请返利比例 (%)</label>
                  <Input defaultValue="10" type="number" className="bg-secondary border-border" />
                </div>
                <div>
                  <label className="text-sm font-medium text-foreground mb-2 block">用户最大设备数</label>
                  <Input defaultValue="5" type="number" className="bg-secondary border-border" />
                </div>
              </div>

              <div className="flex justify-end pt-4">
                <Button className="bg-primary text-primary-foreground hover:bg-primary/90">保存用户设置</Button>
              </div>
            </div>
          </SectionCard>
        </div>
      )}
    </div>
  )
}
