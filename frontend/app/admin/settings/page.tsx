"use client"

import { useEffect, useMemo, useState } from "react"
import type { ReactNode } from "react"
import { Bell, CreditCard, Eye, EyeOff, FileText, Globe, Mail, Save, Send, Settings, Shield } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { toast } from "sonner"
import {
  adminCreatePaymentProvider,
  adminGetPaymentProviders,
  adminGetSettings,
  adminSendTestEmail,
  adminUpdatePaymentProvider,
  adminUpdateSettings,
} from "@/lib/api"

type SettingMap = Record<string, string>
type TabId = "basic" | "subscription" | "payment" | "email" | "seo" | "security"

const defaults: SettingMap = {
  site_name: "Zboard",
  site_url: "",
  subscription_name: "Zboard",
  subscription_domain: "",
  backup_subscription_domain: "",
  support_email: "",
  support_telegram: "",
  support_discord: "",
  default_language: "auto",
  allow_register: "1",
  require_email_verify: "0",
  trial_traffic_gb: "0",
  trial_days: "0",
  user_default_device_limit: "3",
  clash_enabled: "1",
  singbox_enabled: "1",
  v2rayn_enabled: "1",
  subscription_expire_reminder_enabled: "1",
  traffic_alert_enabled: "1",
  renewal_success_email_enabled: "1",
  new_order_notify_enabled: "1",
  reminder_days_before: "3",
  traffic_alert_threshold: "80",
  reminder_interval_hours: "24",
  reminder_max_count: "3",
  smtp_host: "",
  smtp_port: "587",
  smtp_user: "",
  smtp_pass: "",
  smtp_from_name: "Zboard",
  smtp_from_email: "",
  smtp_encryption: "starttls",
  smtp_auth_enabled: "1",
  smtp_ssl_verify_enabled: "1",
  email_template_register_subject: "[{{site_name}}] 您的验证码是 {{code}}",
  email_template_register_body:
    '<!DOCTYPE html>\n<html>\n<body style="font-family: sans-serif;">\n<h2>验证码</h2>\n<p>您好，您的验证码是：</p>\n<p style="font-size: 24px; font-weight: bold; color: #6366f1;">{{code}}</p>\n<p>验证码有效期为 10 分钟。</p>\n<p>如果不是本人操作，请忽略此邮件。</p>\n</body>\n</html>',
  email_template_reset_subject: "[{{site_name}}] 密码重置验证码 {{code}}",
  email_template_reset_body:
    '<!DOCTYPE html>\n<html>\n<body style="font-family: sans-serif;">\n<h2>密码重置</h2>\n<p>您好，您的重置验证码是：</p>\n<p style="font-size: 24px; font-weight: bold; color: #6366f1;">{{code}}</p>\n<p>验证码有效期为 10 分钟。</p>\n<p>如果不是本人操作，请立即检查账号安全。</p>\n</body>\n</html>',
  email_template_expire_subject: "[{{site_name}}] 订阅即将到期提醒",
  email_template_expire_body: "<p>您好，您的订阅将在 {{expire_time}} 到期，请及时续费。</p>",
  email_template_traffic_subject: "[{{site_name}}] 流量使用提醒",
  email_template_traffic_body: "<p>您好，您的订阅流量已接近上限，请留意使用情况。</p>",
  seo_title: "Zboard",
  seo_description: "",
  seo_keywords: "",
  seo_og_image: "",
  seo_twitter_card: "summary_large_image",
  seo_favicon_url: "/favicon.ico",
  seo_canonical_url: "",
  seo_allow_index: "1",
  seo_generate_sitemap: "1",
  seo_structured_data: "1",
  seo_custom_head: "",
  captcha_provider: "none",
  captcha_site_key: "",
  captcha_secret_key: "",
  captcha_enabled_register: "0",
  captcha_enabled_login: "0",
  captcha_enabled_forgot: "0",
  captcha_enabled_ticket: "0",
  turnstile_mode: "managed",
  admin_path: "/admin",
  email_domain_whitelist: "",
}

const tabs: { id: TabId; label: string; icon: any }[] = [
  { id: "basic", label: "注册与用户", icon: Settings },
  { id: "subscription", label: "订阅提醒", icon: Bell },
  { id: "payment", label: "支付设置", icon: CreditCard },
  { id: "email", label: "邮件配置", icon: Mail },
  { id: "seo", label: "站点 SEO", icon: Globe },
  { id: "security", label: "安全验证", icon: Shield },
]

const visibleSettingKeys = Object.keys(defaults)

const emailTemplates = [
  {
    id: "register",
    label: "注册验证码邮件",
    subjectKey: "email_template_register_subject",
    bodyKey: "email_template_register_body",
  },
  {
    id: "reset",
    label: "重置密码邮件",
    subjectKey: "email_template_reset_subject",
    bodyKey: "email_template_reset_body",
  },
  {
    id: "expire",
    label: "订阅到期提醒",
    subjectKey: "email_template_expire_subject",
    bodyKey: "email_template_expire_body",
  },
  {
    id: "traffic",
    label: "流量告急提醒",
    subjectKey: "email_template_traffic_subject",
    bodyKey: "email_template_traffic_body",
  },
] as const

export default function AdminSettingsPage() {
  const [activeTab, setActiveTab] = useState<TabId>("basic")
  const [settings, setSettings] = useState<SettingMap>(defaults)
  const [revealedSecrets, setRevealedSecrets] = useState<Record<string, boolean>>({})
  const [templateId, setTemplateId] = useState<(typeof emailTemplates)[number]["id"]>("register")
  const [testEmail, setTestEmail] = useState("")
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testingEmail, setTestingEmail] = useState(false)

  useEffect(() => {
    adminGetSettings()
      .then((res) => setSettings({ ...defaults, ...(res.settings || {}) }))
      .catch((err) => toast.error(err.message || "加载系统设置失败"))
      .finally(() => setLoading(false))
  }, [])

  const activeTemplate = useMemo(
    () => emailTemplates.find((tpl) => tpl.id === templateId) || emailTemplates[0],
    [templateId],
  )

  const setValue = (key: string, value: string) => {
    setSettings((current) => ({ ...current, [key]: value }))
  }

  const setBool = (key: string, value: boolean) => setValue(key, value ? "1" : "0")
  const bool = (key: string) => settings[key] === "1" || settings[key] === "true"

  const collectSettings = () =>
    Object.fromEntries(visibleSettingKeys.map((key) => [key, settings[key] ?? defaults[key] ?? ""]))

  const persistSettings = async () => {
    await adminUpdateSettings(collectSettings())
  }

  const save = async () => {
    setSaving(true)
    try {
      await persistSettings()
      toast.success("系统设置已保存")
    } catch (err: any) {
      toast.error(err.message || "保存系统设置失败")
    } finally {
      setSaving(false)
    }
  }

  const sendTestEmail = async () => {
    setTestingEmail(true)
    try {
      await persistSettings()
      await adminSendTestEmail(testEmail.trim() || undefined)
      toast.success("测试邮件已发送到当前管理员邮箱")
    } catch (err: any) {
      toast.error(err.message || "发送测试邮件失败")
    } finally {
      setTestingEmail(false)
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">系统设置</h1>
          <p className="text-sm text-muted-foreground mt-1">集中配置注册、订阅提醒、邮件服务和站点展示规则。</p>
        </div>
        <Button onClick={save} disabled={saving}>
          <Save className="w-4 h-4" />
          {saving ? "保存中..." : "保存设置"}
        </Button>
      </div>

      <div className="flex flex-wrap gap-2 rounded-lg border bg-muted/40 p-1">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-2 rounded-md px-3 py-2 text-sm transition ${
              activeTab === tab.id ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <tab.icon className="w-4 h-4" />
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === "basic" && (
        <>
        <SettingsSection icon={Settings} title="注册与用户设置" subtitle="配置用户注册、试用等相关选项">
          <div className="space-y-6">
            <SwitchRow label="允许新用户注册" desc="关闭后仅能通过管理员创建账号" checked={bool("allow_register")} onCheckedChange={(v) => setBool("allow_register", v)} />
            <SwitchRow label="邮箱验证码注册" desc="注册时需要邮箱验证码验证" checked={bool("require_email_verify")} onCheckedChange={(v) => setBool("require_email_verify", v)} />
            <SwitchRow label="新用户赠送试用流量" desc="新注册用户自动获得试用流量" checked={Number(settings.trial_traffic_gb || "0") > 0 || Number(settings.trial_days || "0") > 0} onCheckedChange={(v) => {
              if (v) {
                setSettings((current) => ({
                  ...current,
                  trial_traffic_gb: Number(current.trial_traffic_gb || "0") > 0 ? current.trial_traffic_gb : "2",
                  trial_days: Number(current.trial_days || "0") > 0 ? current.trial_days : "3",
                }))
              } else {
                setSettings((current) => ({ ...current, trial_traffic_gb: "0", trial_days: "0" }))
              }
            }} />

            <Divider />

            <Grid>
              <Field label="试用流量 (GB)">
                <Input type="number" min="0" value={settings.trial_traffic_gb} onChange={(e) => setValue("trial_traffic_gb", e.target.value)} />
              </Field>
              <Field label="试用有效期 (天)">
                <Input type="number" min="0" value={settings.trial_days} onChange={(e) => setValue("trial_days", e.target.value)} />
              </Field>
              <Field label="用户最大设备数">
                <Input type="number" min="1" value={settings.user_default_device_limit} onChange={(e) => setValue("user_default_device_limit", e.target.value)} />
              </Field>
              <Field label="默认语言">
                <Select value={settings.default_language} onValueChange={(v) => setValue("default_language", v)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="auto">跟随浏览器</SelectItem>
                    <SelectItem value="zh-CN">简体中文</SelectItem>
                    <SelectItem value="zh-TW">繁体中文</SelectItem>
                    <SelectItem value="en">English</SelectItem>
                  </SelectContent>
                </Select>
              </Field>
            </Grid>
          </div>
        </SettingsSection>

        <SettingsSection icon={Mail} title="联系方式与支持" subtitle="配置在用户端呈现的客服、社群与联络方式">
          <div className="space-y-6">
            <Grid>
              <Field label="客服支持邮箱" hint="配置后展示在页面工单及服务联系入口">
                <Input value={settings.support_email} onChange={(e) => setValue("support_email", e.target.value)} placeholder="support@example.com" />
              </Field>
              <Field label="Telegram 频道/客服链接" hint="配置后展示在前台页脚的纸飞机按钮上">
                <Input value={settings.support_telegram} onChange={(e) => setValue("support_telegram", e.target.value)} placeholder="https://t.me/your_channel" />
              </Field>
              <Field label="Discord 邀请链接" hint="配置后展示在前台页脚的 Discord 气泡按钮上">
                <Input value={settings.support_discord} onChange={(e) => setValue("support_discord", e.target.value)} placeholder="https://discord.gg/your_server" />
              </Field>
            </Grid>
          </div>
        </SettingsSection>
        </>
      )}

      {activeTab === "subscription" && (
        <SettingsSection icon={Bell} title="订阅提醒设置" subtitle="配置用户订阅到期、流量告急等提醒规则">
          <div className="space-y-6">
            <SwitchRow label="启用到期提醒" desc="订阅即将到期时发送邮件提醒" checked={bool("subscription_expire_reminder_enabled")} onCheckedChange={(v) => setBool("subscription_expire_reminder_enabled", v)} />
            <SwitchRow label="启用流量告急提醒" desc="流量使用达到阈值时发送提醒" checked={bool("traffic_alert_enabled")} onCheckedChange={(v) => setBool("traffic_alert_enabled", v)} />
            <SwitchRow label="启用续费成功通知" desc="用户成功续费后发送确认邮件" checked={bool("renewal_success_email_enabled")} onCheckedChange={(v) => setBool("renewal_success_email_enabled", v)} />
            <SwitchRow label="启用新订单通知" desc="有新订单时通知管理员" checked={bool("new_order_notify_enabled")} onCheckedChange={(v) => setBool("new_order_notify_enabled", v)} />

            <Divider />

            <Grid>
              <Field label="到期提醒提前天数" hint="订阅到期前几天发送提醒">
                <Input type="number" min="1" value={settings.reminder_days_before} onChange={(e) => setValue("reminder_days_before", e.target.value)} />
              </Field>
              <Field label="流量告急阈值 (%)" hint="流量使用达到百分比时提醒">
                <Input type="number" min="1" max="100" value={settings.traffic_alert_threshold} onChange={(e) => setValue("traffic_alert_threshold", e.target.value)} />
              </Field>
              <Field label="提醒发送间隔 (小时)" hint="同一类型提醒的最小间隔">
                <Input type="number" min="1" value={settings.reminder_interval_hours} onChange={(e) => setValue("reminder_interval_hours", e.target.value)} />
              </Field>
              <Field label="最大提醒次数" hint="同一订阅周期内最多提醒次数">
                <Input type="number" min="1" value={settings.reminder_max_count} onChange={(e) => setValue("reminder_max_count", e.target.value)} />
              </Field>
            </Grid>

            <Divider />

            <Grid>
              <Field label="订阅名称">
                <Input value={settings.subscription_name} onChange={(e) => setValue("subscription_name", e.target.value)} />
              </Field>
              <Field label="订阅域名">
                <Input value={settings.subscription_domain} onChange={(e) => setValue("subscription_domain", e.target.value)} placeholder="https://sub.example.com" />
              </Field>
              <Field label="备用订阅域名">
                <Input value={settings.backup_subscription_domain} onChange={(e) => setValue("backup_subscription_domain", e.target.value)} placeholder="https://backup.example.com" />
              </Field>
            </Grid>

            <SwitchRow label="启用 Clash 订阅" checked={bool("clash_enabled")} onCheckedChange={(v) => setBool("clash_enabled", v)} />
            <SwitchRow label="启用 sing-box 订阅" checked={bool("singbox_enabled")} onCheckedChange={(v) => setBool("singbox_enabled", v)} />
            <SwitchRow label="启用 V2rayN 订阅" checked={bool("v2rayn_enabled")} onCheckedChange={(v) => setBool("v2rayn_enabled", v)} />
          </div>
        </SettingsSection>
      )}

      {activeTab === "payment" && <PaymentSettingsPanel />}

      {activeTab === "email" && (
        <div className="space-y-6">
          <SettingsSection icon={Mail} title="SMTP 邮件配置" subtitle="配置邮件发送服务器参数">
            <div className="space-y-6">
              <Grid>
                <Field label="SMTP 服务器">
                  <Input value={settings.smtp_host} onChange={(e) => setValue("smtp_host", e.target.value)} placeholder="smtp.gmail.com" />
                </Field>
                <Field label="SMTP 端口">
                  <Input type="number" min="1" value={settings.smtp_port} onChange={(e) => setValue("smtp_port", e.target.value)} />
                </Field>
                <Field label="SMTP 用户名">
                  <Input value={settings.smtp_user} onChange={(e) => setValue("smtp_user", e.target.value)} placeholder="noreply@example.com" />
                </Field>
                <Field label="SMTP 密码">
                  <SecretInput
                    revealed={!!revealedSecrets.smtp_pass}
                    onToggle={() => setRevealedSecrets((current) => ({ ...current, smtp_pass: !current.smtp_pass }))}
                    value={settings.smtp_pass}
                    onChange={(e) => setValue("smtp_pass", e.target.value)}
                    placeholder="请输入 SMTP 密码"
                  />
                </Field>
                <Field label="发件人名称">
                  <Input value={settings.smtp_from_name} onChange={(e) => setValue("smtp_from_name", e.target.value)} />
                </Field>
                <Field label="发件人邮箱">
                  <Input value={settings.smtp_from_email} onChange={(e) => setValue("smtp_from_email", e.target.value)} placeholder="noreply@example.com" />
                </Field>
              </Grid>

              <RadioGroup
                label="加密方式"
                value={settings.smtp_encryption || "starttls"}
                onValueChange={(v) => {
                  setValue("smtp_encryption", v)
                  if (v === "ssl" && (!settings.smtp_port || settings.smtp_port === "587")) setValue("smtp_port", "465")
                  if (v === "starttls" && (!settings.smtp_port || settings.smtp_port === "465")) setValue("smtp_port", "587")
                }}
                options={[
                  { value: "tls", label: "TLS" },
                  { value: "ssl", label: "SSL" },
                  { value: "starttls", label: "STARTTLS" },
                  { value: "none", label: "无加密" },
                ]}
              />

              <SwitchRow label="启用 SMTP 认证" desc="使用用户名密码进行 SMTP 认证" checked={bool("smtp_auth_enabled")} onCheckedChange={(v) => setBool("smtp_auth_enabled", v)} />
              <SwitchRow label="验证 SSL 证书" desc="验证服务器 SSL 证书有效性" checked={bool("smtp_ssl_verify_enabled")} onCheckedChange={(v) => setBool("smtp_ssl_verify_enabled", v)} />

              <Grid>
                <Field label="测试收件邮箱" hint="留空时默认发送到当前管理员邮箱">
                  <Input type="email" value={testEmail} onChange={(e) => setTestEmail(e.target.value)} placeholder="tester@example.com" />
                </Field>
              </Grid>

              <div className="flex flex-wrap justify-end gap-3">
                <Button variant="outline" onClick={sendTestEmail} disabled={testingEmail}>
                  <Send className="w-4 h-4" />
                  {testingEmail ? "发送中..." : "发送测试邮件"}
                </Button>
                <Button onClick={save} disabled={saving}>
                  <Save className="w-4 h-4" />
                  保存邮件配置
                </Button>
              </div>
            </div>
          </SettingsSection>

          <SettingsSection icon={FileText} title="邮件模板设置" subtitle="自定义各类邮件通知的模板内容">
            <div className="space-y-4">
              <Field label="选择模板">
                <Select value={templateId} onValueChange={(v) => setTemplateId(v as typeof templateId)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {emailTemplates.map((tpl) => (
                      <SelectItem key={tpl.id} value={tpl.id}>{tpl.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
              <Field label="邮件主题">
                <Input value={settings[activeTemplate.subjectKey]} onChange={(e) => setValue(activeTemplate.subjectKey, e.target.value)} />
              </Field>
              <Field label="邮件内容（支持 HTML）">
                <Textarea className="min-h-44 font-mono text-sm" value={settings[activeTemplate.bodyKey]} onChange={(e) => setValue(activeTemplate.bodyKey, e.target.value)} />
              </Field>
              <p className="text-xs text-muted-foreground">
                可用变量：{"{{code}}"} 验证码、{"{{username}}"} 用户名、{"{{email}}"} 邮箱、{"{{expire_time}}"} 过期时间、{"{{site_name}}"} 站点名称
              </p>
              <div className="flex justify-end">
                <Button onClick={save} disabled={saving}>
                  <Save className="w-4 h-4" />
                  保存模板
                </Button>
              </div>
            </div>
          </SettingsSection>
        </div>
      )}

      {activeTab === "seo" && (
        <SettingsSection icon={Globe} title="站点 SEO" subtitle="配置搜索引擎与社交媒体展示信息">
          <div className="space-y-6">
            <Grid>
              <Field label="站点名称">
                <Input value={settings.site_name} onChange={(e) => setValue("site_name", e.target.value)} />
              </Field>
              <Field label="站点地址">
                <Input value={settings.site_url} onChange={(e) => setValue("site_url", e.target.value)} placeholder="https://zboard.io" />
              </Field>
              <Field label="SEO 标题">
                <Input value={settings.seo_title} onChange={(e) => setValue("seo_title", e.target.value)} />
              </Field>
              <Field label="关键词">
                <Input value={settings.seo_keywords} onChange={(e) => setValue("seo_keywords", e.target.value)} placeholder="proxy, subscription, zboard" />
              </Field>
            </Grid>
            <Field label="SEO 描述">
              <Textarea value={settings.seo_description} onChange={(e) => setValue("seo_description", e.target.value)} />
            </Field>
            <Grid>
              <Field label="社交媒体图片 URL" hint="社交媒体分享时显示的图片">
                <Input value={settings.seo_og_image} onChange={(e) => setValue("seo_og_image", e.target.value)} placeholder="https://zboard.io/og-image.png" />
              </Field>
              <Field label="Twitter Card 类型">
                <Select value={settings.seo_twitter_card} onValueChange={(v) => setValue("seo_twitter_card", v)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="summary_large_image">summary_large_image</SelectItem>
                    <SelectItem value="summary">summary</SelectItem>
                  </SelectContent>
                </Select>
              </Field>
              <Field label="Favicon URL">
                <Input value={settings.seo_favicon_url} onChange={(e) => setValue("seo_favicon_url", e.target.value)} placeholder="/favicon.ico" />
              </Field>
              <Field label="规范链接 (Canonical URL)">
                <Input value={settings.seo_canonical_url} onChange={(e) => setValue("seo_canonical_url", e.target.value)} placeholder="https://zboard.io" />
              </Field>
            </Grid>

            <Divider />

            <SwitchRow label="允许搜索引擎索引" desc="在 robots.txt 中允许爬虫抓取" checked={bool("seo_allow_index")} onCheckedChange={(v) => setBool("seo_allow_index", v)} />
            <SwitchRow label="生成 Sitemap" desc="自动生成并更新站点地图" checked={bool("seo_generate_sitemap")} onCheckedChange={(v) => setBool("seo_generate_sitemap", v)} />
            <SwitchRow label="启用结构化数据" desc="添加 JSON-LD 结构化数据标记" checked={bool("seo_structured_data")} onCheckedChange={(v) => setBool("seo_structured_data", v)} />

            <Divider />

            <Field label="自定义 Head 代码">
              <Textarea className="min-h-32 font-mono text-sm" value={settings.seo_custom_head} onChange={(e) => setValue("seo_custom_head", e.target.value)} placeholder="<!-- Google Analytics -->" />
            </Field>
          </div>
        </SettingsSection>
      )}

      {activeTab === "security" && (
        <SettingsSection icon={Shield} title="安全验证设置" subtitle="配置公开页面的人机验证和后台安全项">
          <div className="space-y-6">
            <CaptchaPanel
              provider={settings.captcha_provider || "none"}
              onProviderChange={(v) => setValue("captcha_provider", v)}
              siteKey={settings.captcha_site_key}
              onSiteKeyChange={(v) => setValue("captcha_site_key", v)}
              secretKey={settings.captcha_secret_key}
              onSecretKeyChange={(v) => setValue("captcha_secret_key", v)}
              turnstileMode={settings.turnstile_mode || "managed"}
              onTurnstileModeChange={(v) => setValue("turnstile_mode", v)}
              enabledRegister={bool("captcha_enabled_register")}
              enabledLogin={bool("captcha_enabled_login")}
              enabledForgot={bool("captcha_enabled_forgot")}
              enabledTicket={bool("captcha_enabled_ticket")}
              onToggle={(scene, v) => setBool(`captcha_enabled_${scene}`, v)}
            />

            <Divider />

            <Grid>
              <Field label="后台路径" hint="用于管理端入口提示；实际路由仍以部署配置为准">
                <Input value={settings.admin_path} onChange={(e) => setValue("admin_path", e.target.value)} />
              </Field>
            </Grid>
            <Field label="邮箱域名白名单" hint="支持英文逗号、中文逗号、空格或换行分隔；留空表示不限制邮箱域名">
              <Textarea
                value={settings.email_domain_whitelist}
                onChange={(e) => setValue("email_domain_whitelist", e.target.value)}
                placeholder="gmail.com,qq.com,163.com,yahoo.com,sina.com,126.com,outlook.com,yeah.net,foxmail.com"
              />
            </Field>
          </div>
        </SettingsSection>
      )}
    </div>
  )
}

function SettingsSection({ icon: Icon, title, subtitle, children }: { icon: any; title: string; subtitle: string; children: ReactNode }) {
  return (
    <section className="overflow-hidden rounded-xl border bg-card">
      <div className="flex items-center gap-4 border-b px-5 py-4">
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
          <Icon className="w-6 h-6" />
        </div>
        <div>
          <h2 className="text-lg font-semibold">{title}</h2>
          <p className="text-sm text-muted-foreground">{subtitle}</p>
        </div>
      </div>
      <div className="p-5">{children}</div>
    </section>
  )
}

function Grid({ children }: { children: ReactNode }) {
  return <div className="grid grid-cols-1 gap-4 md:grid-cols-2">{children}</div>
}

function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}

function SecretInput({
  value,
  onChange,
  placeholder,
  revealed,
  onToggle,
}: {
  value: string
  onChange: (event: React.ChangeEvent<HTMLInputElement>) => void
  placeholder?: string
  revealed?: boolean
  onToggle?: () => void
}) {
  const [localRevealed, setLocalRevealed] = useState(false)
  const isRevealed = revealed ?? localRevealed
  const toggle = onToggle ?? (() => setLocalRevealed((current) => !current))
  return (
    <div className="relative">
      <Input
        type={isRevealed ? "text" : "password"}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        autoComplete="new-password"
        className="pr-10"
      />
      <button
        type="button"
        onClick={toggle}
        className="absolute inset-y-0 right-0 inline-flex w-10 items-center justify-center text-muted-foreground transition hover:text-foreground"
        title={isRevealed ? "隐藏" : "显示"}
        aria-label={isRevealed ? "隐藏" : "显示"}
      >
        {isRevealed ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
      </button>
    </div>
  )
}

function Divider() {
  return <div className="h-px bg-border" />
}

function SwitchRow({ label, desc, checked, onCheckedChange }: {
  label: string
  desc?: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div>
        <div className="text-sm font-medium">{label}</div>
        {desc && <div className="text-xs text-muted-foreground mt-1">{desc}</div>}
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  )
}

function RadioGroup({
  label,
  value,
  onValueChange,
  options,
}: {
  label: string
  value: string
  onValueChange: (value: string) => void
  options: { value: string; label: string }[]
}) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <div className="flex flex-wrap gap-3">
        {options.map((option) => {
          const active = value === option.value
          return (
            <button
              key={option.value}
              type="button"
              onClick={() => onValueChange(option.value)}
              className={`inline-flex items-center gap-2 rounded-lg border px-4 py-2 text-sm transition ${
                active ? "border-primary bg-primary/5 text-primary" : "bg-background hover:border-foreground/30"
              }`}
            >
              <span className={`inline-flex h-4 w-4 items-center justify-center rounded-full border ${active ? "border-primary" : "border-muted-foreground/40"}`}>
                {active && <span className="h-2 w-2 rounded-full bg-primary" />}
              </span>
              {option.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}

type PaymentProviderType = "epay" | "stripe" | "paypal" | "nowpayments" | "creem"

type PaymentProviderForm = {
  id?: number
  name: string
  displayName: string
  providerType: PaymentProviderType
  enabled: boolean
  sort: number
  config: Record<string, string>
}

const paymentProviderTemplates: {
  name: string
  displayName: string
  providerType: PaymentProviderType
  description: string
  sort: number
  fields: { key: string; label: string; placeholder?: string; type?: string; hint?: string }[]
}[] = [
  {
    name: "epay",
    displayName: "易支付",
    providerType: "epay",
    description: "适合支付宝、微信、QQ 钱包等聚合支付网关。",
    sort: 10,
    fields: [
      { key: "api_url", label: "网关地址", placeholder: "https://pay.example.com" },
      { key: "pid", label: "商户 ID", placeholder: "1001" },
      { key: "secret_key", label: "商户密钥", type: "password" },
    ],
  },
  {
    name: "stripe",
    displayName: "Stripe",
    providerType: "stripe",
    description: "使用 Stripe Checkout 创建收银台链接，并通过 webhook 确认支付。",
    sort: 20,
    fields: [
      { key: "secret_key", label: "Secret Key", type: "password", placeholder: "sk_live_..." },
      { key: "webhook_secret", label: "Webhook Secret", type: "password", placeholder: "whsec_..." },
      { key: "api_url", label: "API 地址", placeholder: "https://api.stripe.com", hint: "通常保持默认，测试代理或私有网关时再修改。" },
    ],
  },
  {
    name: "paypal",
    displayName: "PayPal",
    providerType: "paypal",
    description: "使用 Orders v2 创建订单，用户批准返回后自动 capture。",
    sort: 30,
    fields: [
      { key: "client_id", label: "Client ID", placeholder: "PayPal REST App Client ID" },
      { key: "client_secret", label: "Client Secret", type: "password" },
      { key: "webhook_id", label: "Webhook ID", placeholder: "WH-...", hint: "建议配置，用于异步回调校验；只靠返回 capture 时可先留空。" },
      { key: "api_url", label: "API 地址", placeholder: "https://api-m.paypal.com / https://api-m.sandbox.paypal.com" },
    ],
  },
  {
    name: "nowpayments",
    displayName: "NOWPayments",
    providerType: "nowpayments",
    description: "加密货币支付，默认使用 USDT TRC20，可通过付款类型扩展币种。",
    sort: 40,
    fields: [
      { key: "api_key", label: "API Key", type: "password" },
      { key: "ipn_secret", label: "IPN Secret", type: "password" },
      { key: "api_url", label: "API 地址", placeholder: "https://api.nowpayments.io" },
    ],
  },
  {
    name: "creem",
    displayName: "Creem.io",
    providerType: "creem",
    description: "使用 Creem Checkout 创建支付链接，并通过 webhook 完成订阅激活。",
    sort: 50,
    fields: [
      { key: "api_key", label: "API Key", type: "password" },
      { key: "webhook_secret", label: "Webhook Secret", type: "password" },
      { key: "api_url", label: "API 地址", placeholder: "https://api.creem.io" },
    ],
  },
]

function PaymentSettingsPanel() {
  const [forms, setForms] = useState<Record<string, PaymentProviderForm>>({})
  const [revealedSecrets, setRevealedSecrets] = useState<Record<string, boolean>>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    try {
      const res = await adminGetPaymentProviders()
      const rows = res.items || []
      const next: Record<string, PaymentProviderForm> = {}
      for (const tpl of paymentProviderTemplates) {
        const row = rows.find((item: any) => item.name === tpl.name || item.provider_type === tpl.providerType)
        next[tpl.name] = {
          id: row?.id,
          name: row?.name || tpl.name,
          displayName: row?.display_name || tpl.displayName,
          providerType: tpl.providerType,
          enabled: row ? Number(row.enabled) === 1 : false,
          sort: row?.sort ?? tpl.sort,
          config: { ...defaultPaymentConfig(tpl.providerType), ...parseConfig(row?.config_json) },
        }
      }
      setForms(next)
    } catch (err: any) {
      setNotice(err?.message || "加载支付设置失败")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const updateForm = (name: string, patch: Partial<PaymentProviderForm>) => {
    setForms((current) => ({ ...current, [name]: { ...current[name], ...patch } }))
  }

  const updateConfig = (name: string, key: string, value: string) => {
    setForms((current) => ({
      ...current,
      [name]: {
        ...current[name],
        config: { ...current[name].config, [key]: value },
      },
    }))
  }

  const saveProvider = async (tpl: (typeof paymentProviderTemplates)[number]) => {
    const form = forms[tpl.name]
    if (!form) return
    setSaving(tpl.name)
    setNotice(null)
    try {
      const payload = {
        name: tpl.name,
        display_name: form.displayName || tpl.displayName,
        provider_type: tpl.providerType,
        config_json: JSON.stringify(compactConfig(form.config)),
        enabled: form.enabled ? 1 : 0,
        sort: form.sort,
      }
      if (form.id) {
        await adminUpdatePaymentProvider(form.id, payload)
      } else {
        await adminCreatePaymentProvider(payload)
      }
      setNotice(`${tpl.displayName} 支付配置已保存`)
      await load()
    } catch (err: any) {
      setNotice(err?.message || "保存支付配置失败")
    } finally {
      setSaving(null)
    }
  }

  if (loading) {
    return <div className="rounded-xl border bg-card p-6 text-sm text-muted-foreground">正在加载支付设置...</div>
  }

  return (
    <SettingsSection icon={CreditCard} title="支付设置" subtitle="配置用户购买套餐和重置流量时可用的支付渠道">
      <div className="space-y-5">
        {notice && <div className="rounded-lg border bg-muted/50 px-4 py-3 text-sm text-muted-foreground">{notice}</div>}
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          {paymentProviderTemplates.map((tpl) => {
            const form = forms[tpl.name]
            if (!form) return null
            return (
              <div key={tpl.name} className="rounded-lg border bg-background p-4">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="text-base font-semibold">{tpl.displayName}</h3>
                      <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">{tpl.providerType}</span>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">{tpl.description}</p>
                  </div>
                  <Switch checked={form.enabled} onCheckedChange={(checked) => updateForm(tpl.name, { enabled: checked })} />
                </div>

                <div className="mt-4 space-y-4">
                  <Grid>
                    <Field label="显示名称">
                      <Input value={form.displayName} onChange={(e) => updateForm(tpl.name, { displayName: e.target.value })} />
                    </Field>
                    <Field label="排序">
                      <Input type="number" value={form.sort} onChange={(e) => updateForm(tpl.name, { sort: Number(e.target.value) || tpl.sort })} />
                    </Field>
                  </Grid>
                  <Grid>
                    {tpl.fields.map((field) => (
                      <Field key={field.key} label={field.label} hint={field.hint}>
                        {field.type === "password" ? (
                          <SecretInput
                            revealed={!!revealedSecrets[`${tpl.name}.${field.key}`]}
                            onToggle={() =>
                              setRevealedSecrets((current) => ({
                                ...current,
                                [`${tpl.name}.${field.key}`]: !current[`${tpl.name}.${field.key}`],
                              }))
                            }
                            value={form.config[field.key] || ""}
                            placeholder={field.placeholder}
                            onChange={(e) => updateConfig(tpl.name, field.key, e.target.value)}
                          />
                        ) : (
                          <Input
                            value={form.config[field.key] || ""}
                            placeholder={field.placeholder}
                            onChange={(e) => updateConfig(tpl.name, field.key, e.target.value)}
                          />
                        )}
                      </Field>
                    ))}
                  </Grid>
                </div>

                <div className="mt-4 flex justify-end">
                  <Button onClick={() => saveProvider(tpl)} disabled={saving === tpl.name}>
                    <Save className="w-4 h-4" />
                    {saving === tpl.name ? "保存中..." : "保存渠道"}
                  </Button>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </SettingsSection>
  )
}

function defaultPaymentConfig(type: PaymentProviderType): Record<string, string> {
  switch (type) {
    case "stripe":
      return { api_url: "https://api.stripe.com", secret_key: "", webhook_secret: "" }
    case "paypal":
      return { api_url: "https://api-m.paypal.com", client_id: "", client_secret: "", webhook_id: "" }
    case "nowpayments":
      return { api_url: "https://api.nowpayments.io", api_key: "", ipn_secret: "" }
    case "creem":
      return { api_url: "https://api.creem.io", api_key: "", webhook_secret: "" }
    case "epay":
    default:
      return { api_url: "", pid: "", secret_key: "" }
  }
}

function parseConfig(raw?: string): Record<string, string> {
  if (!raw) return {}
  try {
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {}
    return Object.fromEntries(Object.entries(parsed).map(([key, value]) => [key, String(value ?? "")]))
  } catch {
    return {}
  }
}

function compactConfig(config: Record<string, string>) {
  return Object.fromEntries(Object.entries(config).map(([key, value]) => [key, String(value ?? "").trim()]))
}

type CaptchaProvider = "none" | "turnstile" | "recaptcha" | "hcaptcha"

const providerCards: { id: CaptchaProvider; title: string; subtitle: string }[] = [
  { id: "none", title: "不启用", subtitle: "关闭人机验证" },
  { id: "turnstile", title: "Turnstile", subtitle: "Cloudflare，推荐" },
  { id: "recaptcha", title: "reCAPTCHA", subtitle: "Google v2/v3" },
  { id: "hcaptcha", title: "hCaptcha", subtitle: "隐私友好" },
]

const turnstileModes = [
  { id: "managed", title: "托管模式", subtitle: "自动选择最佳方式" },
  { id: "non-interactive", title: "非交互式", subtitle: "尽量无感验证" },
  { id: "invisible", title: "隐式验证", subtitle: "提交时触发" },
]

function CaptchaPanel(props: {
  provider: string
  onProviderChange: (v: string) => void
  siteKey: string
  onSiteKeyChange: (v: string) => void
  secretKey: string
  onSecretKeyChange: (v: string) => void
  turnstileMode: string
  onTurnstileModeChange: (v: string) => void
  enabledRegister: boolean
  enabledLogin: boolean
  enabledForgot: boolean
  enabledTicket: boolean
  onToggle: (scene: "register" | "login" | "forgot" | "ticket", v: boolean) => void
}) {
  const provider = (props.provider || "none") as CaptchaProvider
  return (
    <div className="space-y-5">
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {providerCards.map((card) => {
          const active = provider === card.id
          return (
            <button
              key={card.id}
              type="button"
              onClick={() => props.onProviderChange(card.id)}
              className={`flex flex-col items-start gap-1 rounded-lg border bg-background p-3 text-left transition ${
                active ? "border-primary ring-2 ring-primary/30 shadow-sm" : "hover:border-foreground/30"
              }`}
            >
              <div className="text-sm font-semibold">{card.title}</div>
              <div className="text-xs text-muted-foreground">{card.subtitle}</div>
            </button>
          )
        })}
      </div>

      {provider !== "none" && (
        <div className="space-y-5 rounded-lg border bg-background p-5">
          <Grid>
            <Field label="Site Key">
              <Input value={props.siteKey} onChange={(e) => props.onSiteKeyChange(e.target.value)} placeholder="0x4AAAAAAA..." />
            </Field>
            <Field label="Secret Key">
              <SecretInput value={props.secretKey} onChange={(e) => props.onSecretKeyChange(e.target.value)} placeholder="0x4AAAAAAA..." />
            </Field>
          </Grid>

          {provider === "turnstile" && (
            <div className="space-y-2">
              <Label>验证模式</Label>
              <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
                {turnstileModes.map((mode) => {
                  const active = props.turnstileMode === mode.id
                  return (
                    <button
                      key={mode.id}
                      type="button"
                      onClick={() => props.onTurnstileModeChange(mode.id)}
                      className={`rounded-lg border bg-background p-3 text-left transition ${
                        active ? "border-primary ring-2 ring-primary/30" : "hover:border-foreground/30"
                      }`}
                    >
                      <div className="text-sm font-medium">{mode.title}</div>
                      <div className="text-xs text-muted-foreground">{mode.subtitle}</div>
                    </button>
                  )
                })}
              </div>
            </div>
          )}

          <div className="space-y-3">
            <div className="text-sm font-medium">启用页面</div>
            <ToggleRow label="注册页" checked={props.enabledRegister} onChange={(v) => props.onToggle("register", v)} />
            <ToggleRow label="登录页" checked={props.enabledLogin} onChange={(v) => props.onToggle("login", v)} />
            <ToggleRow label="找回密码" checked={props.enabledForgot} onChange={(v) => props.onToggle("forgot", v)} />
            <ToggleRow label="提交工单" checked={props.enabledTicket} onChange={(v) => props.onToggle("ticket", v)} />
          </div>
        </div>
      )}
    </div>
  )
}

function ToggleRow({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="text-sm">{label}</div>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  )
}
