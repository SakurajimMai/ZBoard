"use client"

import { FormEvent, ReactNode, useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { CircleUserRound, KeyRound, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { changeMyPassword, deleteMyAccount, getMe, logout } from "@/lib/api"
import { useConfirm } from "@/components/confirm-dialog"
import { useI18n } from "@/lib/i18n/context"
import { dashboardCopy } from "@/lib/i18n/dashboard"

type UserProfile = {
  email?: string
  status?: string
}

export default function SettingsPage() {
  const router = useRouter()
  const confirm = useConfirm()
  const { locale } = useI18n()
  const d = dashboardCopy(locale)
  const [user, setUser] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [passwordForm, setPasswordForm] = useState({
    currentPassword: "",
    newPassword: "",
    confirmPassword: "",
  })
  const [deletePassword, setDeletePassword] = useState("")
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const [savingPassword, setSavingPassword] = useState(false)
  const [deletingAccount, setDeletingAccount] = useState(false)

  useEffect(() => {
    getMe()
      .then((res) => setUser(res.user))
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  const clearNotice = () => {
    setMessage("")
    setError("")
  }

  const handlePasswordSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    clearNotice()

    if (passwordForm.newPassword.length < 6) {
      setError(d.settings.passwordMinLength)
      return
    }
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setError(d.settings.passwordMismatch)
      return
    }

    setSavingPassword(true)
    try {
      await changeMyPassword(passwordForm.currentPassword, passwordForm.newPassword)
      setPasswordForm({ currentPassword: "", newPassword: "", confirmPassword: "" })
      setMessage(d.settings.passwordUpdated)
    } catch (err: any) {
      setError(err.message || d.settings.passwordUpdateFailed)
    } finally {
      setSavingPassword(false)
    }
  }

  const handleDeleteAccount = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    clearNotice()

    if (!deletePassword) {
      setError(d.settings.deletePasswordRequired)
      return
    }
    if (!(await confirm({ title: d.settings.deleteAccount, description: d.settings.deleteConfirm, destructive: true, confirmText: d.settings.deleteAccount }))) {
      return
    }

    setDeletingAccount(true)
    try {
      await deleteMyAccount(deletePassword)
      logout()
      router.replace("/login")
    } catch (err: any) {
      setError(err.message || d.settings.deleteFailed)
      setDeletingAccount(false)
    }
  }

  if (loading) {
    return <div className="p-8 text-muted-foreground">{d.common.loading}</div>
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">{d.settings.title}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{d.settings.subtitle}</p>
      </div>

      {(message || error) && (
        <div
          className={`rounded-lg border px-4 py-3 text-sm ${
            error
              ? "border-destructive/30 bg-destructive/10 text-destructive"
              : "border-emerald-500/30 bg-emerald-500/10 text-emerald-700"
          }`}
        >
          {error || message}
        </div>
      )}

      <section className="rounded-xl border border-border bg-card p-6">
        <div className="mb-5 flex items-center gap-4">
          <CircleUserRound className="h-12 w-12 flex-shrink-0 text-foreground/80" strokeWidth={1.8} />
          <div>
            <h2 className="font-semibold text-foreground">{d.settings.basicInfo}</h2>
            <p className="text-sm text-muted-foreground">{d.settings.currentAccount}</p>
          </div>
        </div>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Field label={d.settings.email}>
            <Input value={user?.email || ""} readOnly className="bg-secondary border-border" />
          </Field>
          <Field label={d.settings.accountStatus}>
            <Input value={user?.status === "active" ? d.settings.statusNormal : user?.status || "-"} readOnly className="bg-secondary border-border" />
          </Field>
        </div>
      </section>

      <section className="rounded-xl border border-border bg-card p-6">
        <div className="mb-5 flex items-center gap-3">
          <KeyRound className="h-5 w-5 text-primary" />
          <h2 className="font-semibold text-foreground">{d.settings.changePassword}</h2>
        </div>
        <form onSubmit={handlePasswordSubmit} className="space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <Field label={d.settings.currentPassword}>
              <Input
                type="password"
                value={passwordForm.currentPassword}
                onChange={(e) => setPasswordForm((current) => ({ ...current, currentPassword: e.target.value }))}
                autoComplete="current-password"
                required
              />
            </Field>
            <Field label={d.settings.newPassword}>
              <Input
                type="password"
                value={passwordForm.newPassword}
                onChange={(e) => setPasswordForm((current) => ({ ...current, newPassword: e.target.value }))}
                autoComplete="new-password"
                required
              />
            </Field>
            <Field label={d.settings.confirmNewPassword}>
              <Input
                type="password"
                value={passwordForm.confirmPassword}
                onChange={(e) => setPasswordForm((current) => ({ ...current, confirmPassword: e.target.value }))}
                autoComplete="new-password"
                required
              />
            </Field>
          </div>
          <Button type="submit" disabled={savingPassword}>
            {savingPassword ? d.settings.updating : d.settings.updatePassword}
          </Button>
        </form>
      </section>

      <section className="rounded-xl border border-destructive/40 bg-destructive/5 p-6">
        <div className="mb-5 flex items-center gap-3">
          <Trash2 className="h-5 w-5 text-destructive" />
          <div>
            <h2 className="font-semibold text-foreground">{d.settings.dangerZone}</h2>
            <p className="text-sm text-muted-foreground">{d.settings.dangerDesc}</p>
          </div>
        </div>
        <form onSubmit={handleDeleteAccount} className="space-y-4">
          <Field label={d.settings.currentPassword}>
            <Input
              type="password"
              value={deletePassword}
              onChange={(e) => setDeletePassword(e.target.value)}
              autoComplete="current-password"
              required
            />
          </Field>
          <Button type="submit" variant="destructive" disabled={deletingAccount}>
            {deletingAccount ? d.settings.deleting : d.settings.deleteAccount}
          </Button>
        </form>
      </section>
    </div>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}
