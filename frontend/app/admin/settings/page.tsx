"use client"

import { useEffect, useMemo, useState } from "react"
import type { ReactNode } from "react"
import { Bell, Globe, Mail, Save, Settings, Shield } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { adminGetSettings, adminUpdateSettings } from "@/lib/api"

type SettingMap = Record<string, string>

const defaults: SettingMap = {
  site_name: "Zboard",
  site_url: "",
  subscription_name: "Zboard",
  subscription_domain: "",
  support_email: "",
  support_telegram: "",
  default_language: "zh-CN",
  allow_register: "1",
  require_email_verify: "0",
  trial_traffic_gb: "0",
  trial_days: "0",
  user_default_device_limit: "3",
  clash_enabled: "1",
  singbox_enabled: "1",
  v2rayn_enabled: "1",
  smtp_host: "",
  smtp_port: "587",
  smtp_user: "",
  smtp_pass: "",
  smtp_from_name: "",
  smtp_from_email: "",
  smtp_encryption: "starttls",
  captcha_provider: "none",
  captcha_site_key: "",
  captcha_secret_key: "",
  admin_path: "/admin",
  email_domain_whitelist: "",
  seo_title: "Zboard",
  seo_description: "",
  seo_keywords: "",
}

const tabs = [
  { id: "basic", label: "基础设置", icon: Settings },
  { id: "security", label: "安全设置", icon: Shield },
  { id: "subscription", label: "订阅设置", icon: Bell },
  { id: "email", label: "邮件配置", icon: Mail },
  { id: "seo", label: "站点 SEO", icon: Globe },
] as const

const visibleSettingKeys = Object.keys(defaults)

export default function AdminSettingsPage() {
  const [activeTab, setActiveTab] = useState<(typeof tabs)[number]["id"]>("basic")
  const [settings, setSettings] = useState<SettingMap>(defaults)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    adminGetSettings()
      .then((res) => setSettings({ ...defaults, ...(res.settings || {}) }))
      .catch((err) => alert(err.message || "加载系统设置失败"))
      .finally(() => setLoading(false))
  }, [])

  const activeLabel = useMemo(() => tabs.find((tab) => tab.id === activeTab)?.label || "系统设置", [activeTab])

  const setValue = (key: string, value: string) => {
    setSettings((current) => ({ ...current, [key]: value }))
  }

  const setBool = (key: string, value: boolean) => {
    setValue(key, value ? "1" : "0")
  }

  const bool = (key: string) => settings[key] === "1" || settings[key] === "true"

  const save = async () => {
    setSaving(true)
    try {
      const visibleSettings = Object.fromEntries(
        visibleSettingKeys.map((key) => [key, settings[key] ?? defaults[key] ?? ""]),
      )
      await adminUpdateSettings(visibleSettings)
      alert("系统设置已保存")
    } catch (err: any) {
      alert(err.message || "保存系统设置失败")
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="text-muted-foreground p-8">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">系统设置</h1>
          <p className="text-sm text-muted-foreground mt-1">
            当前页设置会保存到后端配置表；注册开关和邮箱验证要求已接入真实注册流程。
          </p>
        </div>
        <Button onClick={save} disabled={saving}>
          <Save className="w-4 h-4 mr-1" />
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

      <section className="rounded-lg border bg-card">
        <div className="border-b px-5 py-4">
          <h2 className="font-semibold">{activeLabel}</h2>
        </div>
        <div className="p-5">
          {activeTab === "basic" && (
            <Grid>
              <Field label="站点名称">
                <Input value={settings.site_name} onChange={(e) => setValue("site_name", e.target.value)} />
              </Field>
              <Field label="站点地址">
                <Input value={settings.site_url} onChange={(e) => setValue("site_url", e.target.value)} placeholder="https://example.com" />
              </Field>
              <Field label="默认语言">
                <Select value={settings.default_language} onValueChange={(v) => setValue("default_language", v)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="zh-CN">简体中文</SelectItem>
                    <SelectItem value="zh-TW">繁體中文</SelectItem>
                    <SelectItem value="en">English</SelectItem>
                  </SelectContent>
                </Select>
              </Field>
              <Field label="支持邮箱">
                <Input value={settings.support_email} onChange={(e) => setValue("support_email", e.target.value)} />
              </Field>
              <Field label="Telegram">
                <Input value={settings.support_telegram} onChange={(e) => setValue("support_telegram", e.target.value)} />
              </Field>
              <Field label="默认设备数">
                <Input type="number" min="1" value={settings.user_default_device_limit} onChange={(e) => setValue("user_default_device_limit", e.target.value)} />
              </Field>
            </Grid>
          )}

          {activeTab === "security" && (
            <div className="space-y-6">
              <SwitchRow label="允许用户注册" desc="关闭后普通注册和注册验证码都会被拒绝。" checked={bool("allow_register")} onCheckedChange={(v) => setBool("allow_register", v)} />
              <SwitchRow label="强制邮箱验证码注册" desc="开启后 /auth/register 会拒绝普通密码注册，用户必须走验证码注册接口。" checked={bool("require_email_verify")} onCheckedChange={(v) => setBool("require_email_verify", v)} />
              <Grid>
                <Field label="验证码服务">
                  <Select value={settings.captcha_provider} onValueChange={(v) => setValue("captcha_provider", v)}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">不启用</SelectItem>
                      <SelectItem value="turnstile">Cloudflare Turnstile</SelectItem>
                      <SelectItem value="recaptcha">Google reCAPTCHA</SelectItem>
                      <SelectItem value="hcaptcha">hCaptcha</SelectItem>
                    </SelectContent>
                  </Select>
                </Field>
                <Field label="Site Key">
                  <Input value={settings.captcha_site_key} onChange={(e) => setValue("captcha_site_key", e.target.value)} />
                </Field>
                <Field label="Secret Key">
                  <Input type="password" value={settings.captcha_secret_key} onChange={(e) => setValue("captcha_secret_key", e.target.value)} />
                </Field>
                <Field label="后台路径">
                  <Input value={settings.admin_path} onChange={(e) => setValue("admin_path", e.target.value)} />
                </Field>
              </Grid>
              <Field label="邮箱域名白名单">
                <Textarea value={settings.email_domain_whitelist} onChange={(e) => setValue("email_domain_whitelist", e.target.value)} placeholder="每行一个域名，例如 gmail.com" />
              </Field>
            </div>
          )}

          {activeTab === "subscription" && (
            <div className="space-y-6">
              <Grid>
                <Field label="订阅名称">
                  <Input value={settings.subscription_name} onChange={(e) => setValue("subscription_name", e.target.value)} />
                </Field>
                <Field label="订阅域名">
                  <Input value={settings.subscription_domain} onChange={(e) => setValue("subscription_domain", e.target.value)} placeholder="https://sub.example.com" />
                </Field>
                <Field label="试用流量 GB">
                  <Input type="number" min="0" value={settings.trial_traffic_gb} onChange={(e) => setValue("trial_traffic_gb", e.target.value)} />
                </Field>
                <Field label="试用天数">
                  <Input type="number" min="0" value={settings.trial_days} onChange={(e) => setValue("trial_days", e.target.value)} />
                </Field>
              </Grid>
              <SwitchRow label="启用 Clash 订阅" checked={bool("clash_enabled")} onCheckedChange={(v) => setBool("clash_enabled", v)} />
              <SwitchRow label="启用 sing-box 订阅" checked={bool("singbox_enabled")} onCheckedChange={(v) => setBool("singbox_enabled", v)} />
              <SwitchRow label="启用 V2rayN 订阅" checked={bool("v2rayn_enabled")} onCheckedChange={(v) => setBool("v2rayn_enabled", v)} />
            </div>
          )}

          {activeTab === "email" && (
            <Grid>
              <Field label="SMTP Host">
                <Input value={settings.smtp_host} onChange={(e) => setValue("smtp_host", e.target.value)} />
              </Field>
              <Field label="SMTP Port">
                <Input type="number" value={settings.smtp_port} onChange={(e) => setValue("smtp_port", e.target.value)} />
              </Field>
              <Field label="SMTP User">
                <Input value={settings.smtp_user} onChange={(e) => setValue("smtp_user", e.target.value)} />
              </Field>
              <Field label="SMTP Password">
                <Input type="password" value={settings.smtp_pass} onChange={(e) => setValue("smtp_pass", e.target.value)} />
              </Field>
              <Field label="发件人名称">
                <Input value={settings.smtp_from_name} onChange={(e) => setValue("smtp_from_name", e.target.value)} />
              </Field>
              <Field label="发件人邮箱">
                <Input value={settings.smtp_from_email} onChange={(e) => setValue("smtp_from_email", e.target.value)} />
              </Field>
              <Field label="加密方式">
                <Select value={settings.smtp_encryption} onValueChange={(v) => setValue("smtp_encryption", v)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">None</SelectItem>
                    <SelectItem value="starttls">STARTTLS</SelectItem>
                    <SelectItem value="ssl">SSL/TLS</SelectItem>
                  </SelectContent>
                </Select>
              </Field>
            </Grid>
          )}

          {activeTab === "seo" && (
            <div className="space-y-4">
              <Field label="SEO 标题">
                <Input value={settings.seo_title} onChange={(e) => setValue("seo_title", e.target.value)} />
              </Field>
              <Field label="SEO 描述">
                <Textarea value={settings.seo_description} onChange={(e) => setValue("seo_description", e.target.value)} />
              </Field>
              <Field label="关键词">
                <Textarea value={settings.seo_keywords} onChange={(e) => setValue("seo_keywords", e.target.value)} placeholder="用英文逗号分隔" />
              </Field>
            </div>
          )}
        </div>
      </section>
    </div>
  )
}

function Grid({ children }: { children: ReactNode }) {
  return <div className="grid grid-cols-1 md:grid-cols-2 gap-4">{children}</div>
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}

function SwitchRow({ label, desc, checked, onCheckedChange }: {
  label: string
  desc?: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border bg-background p-4">
      <div>
        <div className="text-sm font-medium">{label}</div>
        {desc && <div className="text-xs text-muted-foreground mt-1">{desc}</div>}
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  )
}
