// API base URL.
// 默认走同源(空字符串)+ Next.js rewrites 反代到后端,这样:
//   - 走 CDN 时只需代理 80/443,不必暴露 :3000
//   - 浏览器无跨域,无需 CORS 预检
// 如果显式指定 NEXT_PUBLIC_API_URL 则走绝对地址(用于独立 API 子域名场景)。
export function getApiBase(): string {
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL.replace(/\/+$/, '')
  }
  return ''
}

const API_BASE = getApiBase()

export function normalizeOrigin(value?: string): string {
  const raw = (value || '').trim().replace(/\/+$/, '')
  if (!raw) return ''
  if (/^https?:\/\//i.test(raw)) return raw
  return `https://${raw}`
}

export function buildSubscriptionUrlFromBase(base: string, token: string, target?: string): string {
  const origin = normalizeOrigin(base)
  const suffix = target ? `?target=${encodeURIComponent(target)}` : ''
  return `${origin}/api/sub/${token}${suffix}`
}

export function getSubscriptionBase(settings?: Record<string, string>): string {
  const configured = normalizeOrigin(settings?.subscription_domain)
  if (configured) return configured
  const apiBase = normalizeOrigin(getApiBase())
  if (apiBase) return apiBase
  if (typeof window !== 'undefined') return window.location.origin.replace(/\/+$/, '')
  return ''
}

export function buildSubscriptionUrl(token: string, target?: string, settings?: Record<string, string>): string {
  return buildSubscriptionUrlFromBase(getSubscriptionBase(settings), token, target)
}

class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = typeof window !== 'undefined' ? localStorage.getItem('zboard_token') : null
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  })

  if (!res.ok) {
    const body = await res.json().catch(() => ({ code: 'unknown', message: res.statusText }))
    throw new ApiError(res.status, body.code || 'unknown', body.message || res.statusText)
  }

  return res.json()
}

// ===== Auth =====

export async function sendEmailCode(email: string, purpose: 'register' | 'reset_password', captchaToken?: string) {
  return request<{ ok: boolean }>('/api/v1/auth/send-email-code', {
    method: 'POST',
    body: JSON.stringify({ email, purpose, captcha_token: captchaToken || '' }),
  })
}

export async function registerWithCode(email: string, password: string, code: string, captchaToken?: string) {
  return request<{ user_id: number }>('/api/v1/auth/register-with-code', {
    method: 'POST',
    body: JSON.stringify({ email, password, code, captcha_token: captchaToken || '' }),
  })
}

export async function resetPassword(email: string, newPassword: string, code: string, captchaToken?: string) {
  return request<{ ok: boolean }>('/api/v1/auth/reset-password', {
    method: 'POST',
    body: JSON.stringify({ email, new_password: newPassword, code, captcha_token: captchaToken || '' }),
  })
}

export async function register(email: string, password: string, captchaToken?: string) {
  return request<{ user_id: number }>('/api/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, password, captcha_token: captchaToken || '' }),
  })
}

export async function login(email: string, password: string, captchaToken?: string) {
  const data = await request<{ token: string; user: any }>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password, captcha_token: captchaToken || '' }),
  })
  if (typeof window !== 'undefined') {
    localStorage.setItem('zboard_token', data.token)
  }
  return data
}

export async function adminLogin(email: string, password: string) {
  const data = await request<{ token: string; admin: any }>('/api/admin/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
  if (typeof window !== 'undefined') {
    localStorage.setItem('zboard_admin_token', data.token)
  }
  return data
}

export function logout() {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('zboard_token')
  }
}

export function adminLogout() {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('zboard_admin_token')
  }
}

export function getToken(): string | null {
  if (typeof window === 'undefined') return null
  return localStorage.getItem('zboard_token')
}

export function getAdminToken(): string | null {
  if (typeof window === 'undefined') return null
  return localStorage.getItem('zboard_admin_token')
}

// ===== User =====

export async function getMe() {
  return request<{ user: any }>('/api/v1/me')
}

export async function changeMyPassword(currentPassword: string, newPassword: string) {
  return request<{ ok: boolean }>('/api/v1/me/password', {
    method: 'POST',
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  })
}

export async function deleteMyAccount(password: string) {
  return request<{ ok: boolean }>('/api/v1/me', {
    method: 'DELETE',
    body: JSON.stringify({ password }),
  })
}

export async function getDailyTraffic(days: number = 30) {
  return request<{ items: { day: string; upload: number; download: number; total: number }[]; days: number }>(
    `/api/v1/traffic/daily?days=${days}`,
  )
}

export async function resetMyTraffic() {
  return request<{ order: any }>('/api/v1/traffic/reset', { method: 'POST' })
}

export async function resetMyUUID() {
  return request<{ ok: boolean; client_id: string }>('/api/v1/uuid/reset', { method: 'POST' })
}

export async function getPlans() {
  return request<{ items: any[] }>('/api/v1/plans')
}

export async function createOrder(planId: number, period: 'monthly' | 'quarterly' | 'yearly' = 'monthly') {
  return request<{ order: any; existing: boolean }>('/api/v1/orders', {
    method: 'POST',
    body: JSON.stringify({ plan_id: planId, period }),
  })
}

export async function payOrder(orderNo: string, provider?: string, payType?: string) {
  let path = `/api/v1/orders/${orderNo}/pay`
  const params = new URLSearchParams()
  if (provider) params.set('provider', provider)
  if (payType) params.set('pay_type', payType)
  if (params.toString()) path += `?${params.toString()}`
  return request<{ pay_url: string; order_no: string }>(`${path}`, { method: 'POST' })
}

export async function getSubscription() {
  return request<{ token: string }>('/api/v1/subscription')
}

export async function resetSubscriptionToken() {
  return request<{ token: string }>('/api/v1/subscription/reset-token', { method: 'POST' })
}

export async function getPaymentMethods() {
  return request<{ methods: any[] }>('/api/v1/payment-methods')
}

export async function getTrafficSnapshot() {
  return request<{ snapshot: { upload_total: number; download_total: number; total_used: number; traffic_limit: number } }>('/api/v1/traffic/snapshot')
}

export async function getTrafficLogs(limit?: number) {
  const q = limit ? `?limit=${limit}` : ''
  return request<{ items: { id: number; user_id: number; node_id: number; upload_delta: number; download_delta: number; total_delta: number; reported_at: string; created_at: string }[] }>(`/api/v1/traffic/logs${q}`)
}

export async function getUserNodes() {
  return request<{ items: { id: number; name: string; region: string | null; protocol: string; transport: string; status: string; last_heartbeat_at: string | null }[] }>('/api/v1/nodes')
}

// ===== Admin =====

function adminRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = typeof window !== 'undefined' ? localStorage.getItem('zboard_admin_token') : null
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  return fetch(`${API_BASE}${path}`, { ...options, headers }).then(async (res) => {
    if (!res.ok) {
      const body = await res.json().catch(() => ({ code: 'unknown', message: res.statusText }))
      throw new ApiError(res.status, body.code, body.message)
    }
    return res.json()
  })
}

export async function adminGetMe() {
  return adminRequest<{ admin: any }>('/api/admin/v1/auth/me')
}

type PageQuery = { page?: number; pageSize?: number }
type UserQuery = PageQuery & {
  email?: string
  status?: string
  planId?: string
  expires?: string
  trafficMin?: number | null
  trafficMax?: number | null
}
type PageResponse<T = any> = { items: T[]; page?: number; page_size?: number; total?: number }

function pageQuery(params?: PageQuery & Record<string, any>) {
  if (!params) return ''
  const q = new URLSearchParams()
  if (params.page) q.set('page', String(params.page))
  if (params.pageSize) q.set('page_size', String(params.pageSize))
  for (const [key, value] of Object.entries(params)) {
    if (key === 'page' || key === 'pageSize') continue
    if (value === undefined || value === null || value === '' || value === 'all') continue
    const apiKey = key.replace(/[A-Z]/g, (m) => `_${m.toLowerCase()}`)
    q.set(apiKey, String(value))
  }
  const s = q.toString()
  return s ? `?${s}` : ''
}

export async function adminGetUsers(params?: UserQuery) {
  return adminRequest<PageResponse>(`/api/admin/v1/users${pageQuery(params)}`)
}

export async function adminGetOverview() {
  return adminRequest<{
    users: number
    active_nodes: number
    paid_orders: number
    revenue: string
    revenue_trend: Array<{ month: string; label: string; revenue: number }>
    traffic_trend: Array<{ day: string; label: string; total: number; tb: number }>
  }>('/api/admin/v1/overview')
}

export async function adminCreateUser(data: any) {
  return adminRequest<{ user_id: number }>('/api/admin/v1/users', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export async function adminUpdateUser(id: number, data: any) {
  return adminRequest<{ ok: boolean }>(`/api/admin/v1/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export async function adminBatchUsers(
  action: 'enable' | 'disable' | 'reset_subscription' | 'send_email',
  userIds: number[],
  extra?: { subject?: string; content?: string },
) {
  return adminRequest<{ ok: boolean; count: number }>('/api/admin/v1/users/batch', {
    method: 'POST',
    body: JSON.stringify({ action, user_ids: userIds, ...(extra || {}) }),
  })
}

export async function adminResetUserSubscription(id: number) {
  return adminRequest<{ token: string }>(`/api/admin/v1/users/${id}/reset-subscription`, { method: 'POST' })
}

export async function adminGetUserSubscription(id: number) {
  return adminRequest<{ token: string }>(`/api/admin/v1/users/${id}/subscription`)
}

export async function adminResetUserUUID(id: number) {
  return adminRequest<{ ok: boolean; client_id: string }>(`/api/admin/v1/users/${id}/reset-uuid`, { method: 'POST' })
}

export async function adminResetUserIdentity(id: number) {
  return adminRequest<{ token: string; client_id: string }>(`/api/admin/v1/users/${id}/reset-identity`, { method: 'POST' })
}

export async function adminGetUserOrders(id: number) {
  return adminRequest<{ items: any[] }>(`/api/admin/v1/users/${id}/orders`)
}

export async function adminGetUserTrafficLogs(id: number) {
  return adminRequest<{ items: any[] }>(`/api/admin/v1/users/${id}/traffic-logs`)
}

export async function adminGetOrders(params?: PageQuery) {
  return adminRequest<PageResponse>(`/api/admin/v1/orders${pageQuery(params)}`)
}

export async function adminGetNodes(params?: PageQuery) {
  return adminRequest<PageResponse>(`/api/admin/v1/nodes${pageQuery(params)}`)
}

export async function adminCreateNode(data: any) {
  return adminRequest<{ node_id: number; node_secret: string }>('/api/admin/v1/nodes', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export async function adminUpdateNode(id: number, data: any) {
  return adminRequest<{ ok: boolean }>(`/api/admin/v1/nodes/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export async function adminGenerateRealityConfig(serverName?: string) {
  return adminRequest<{
    reality_public_key: string
    reality_private_key: string
    reality_short_id: string
    reality_server_name: string
    reality_dest: string
  }>('/api/admin/v1/reality/generate', {
    method: 'POST',
    body: JSON.stringify({ server_name: serverName || '' }),
  })
}

export async function adminGetPlans(params?: PageQuery) {
  return adminRequest<PageResponse>(`/api/admin/v1/plans${pageQuery(params)}`)
}

export async function adminCreatePlan(data: any) {
  return adminRequest<{ plan_id: number }>('/api/admin/v1/plans', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export async function adminUpdatePlan(id: number, data: any) {
  return adminRequest<{ ok: boolean }>(`/api/admin/v1/plans/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export async function adminSyncNodeConfig(nodeId: number) {
  return adminRequest<{ task_id: string; version: string }>(`/api/admin/v1/nodes/${nodeId}/sync-config`, {
    method: 'POST',
  })
}

export async function adminSyncAllNodeConfigs() {
  return adminRequest<{
    ok: number
    failed: number
    total: number
    results: { node_id: number; name: string; task_id?: string; version?: string; error?: string }[]
  }>('/api/admin/v1/nodes/sync-config-all', { method: 'POST' })
}

export async function adminGetPaymentProviders() {
  return adminRequest<{ items: any[] }>('/api/admin/v1/payment-providers')
}

export async function adminCreatePaymentProvider(data: any) {
  return adminRequest<{ id: number }>('/api/admin/v1/payment-providers', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export async function adminUpdatePaymentProvider(id: number, data: any) {
  return adminRequest<{ ok: boolean }>(`/api/admin/v1/payment-providers/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export async function adminDeletePaymentProvider(id: number) {
  return adminRequest<{ ok: boolean }>(`/api/admin/v1/payment-providers/${id}`, {
    method: 'DELETE',
  })
}

export async function adminRunMaintenance() {
  return adminRequest<any>('/api/admin/v1/workers/maintenance/run', { method: 'POST' })
}

export async function adminGetAuditLogs() {
  return adminRequest<{ items: any[] }>('/api/admin/v1/audit-logs')
}

export async function adminGetTrafficUsers() {
  return adminRequest<{ items: any[] }>('/api/admin/v1/traffic/users')
}

export async function getPublicSettings() {
  return request<{ settings: Record<string, string> }>('/api/v1/settings')
}

export async function adminGetSettings() {
  return adminRequest<{ settings: Record<string, string> }>('/api/admin/v1/settings')
}

export async function adminUpdateSettings(settings: Record<string, string>) {
  return adminRequest<{ ok: boolean }>('/api/admin/v1/settings', {
    method: 'PUT',
    body: JSON.stringify({ settings }),
  })
}

export async function adminSendTestEmail(email?: string) {
  return adminRequest<{ ok: boolean }>('/api/admin/v1/settings/test-email', {
    method: 'POST',
    body: JSON.stringify({ email: email || '' }),
  })
}

export async function adminGetAnnouncements(params?: PageQuery) {
  return adminRequest<PageResponse>(`/api/admin/v1/announcements${pageQuery(params)}`)
}

export async function adminCreateAnnouncement(data: any) {
  return adminRequest<{ id: number }>('/api/admin/v1/announcements', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export async function adminUpdateAnnouncement(id: number, data: any) {
  return adminRequest<{ ok: boolean }>(`/api/admin/v1/announcements/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export async function adminDeleteAnnouncement(id: number) {
  return adminRequest<{ ok: boolean }>(`/api/admin/v1/announcements/${id}`, {
    method: 'DELETE',
  })
}

export async function getAnnouncements() {
  return request<{ items: any[] }>('/api/v1/announcements')
}

// ===== Tickets (User) =====

export async function getTickets() {
  return request<{ items: any[] }>('/api/v1/tickets')
}

export async function createTicket(subject: string, category: string, content: string, captchaToken?: string) {
  return request<{ ticket_id: number; ticket_no: string }>('/api/v1/tickets', {
    method: 'POST',
    body: JSON.stringify({ subject, category, content, captcha_token: captchaToken || '' }),
  })
}

export async function getTicketDetail(ticketNo: string) {
  return request<{ ticket: any; messages: any[] }>(`/api/v1/tickets/${ticketNo}`)
}

export async function replyTicket(ticketNo: string, content: string) {
  return request<{ ok: boolean }>(`/api/v1/tickets/${ticketNo}/reply`, {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
}

// ===== Notifications =====

export async function getNotifications() {
  return request<{ items: any[]; unread: number }>('/api/v1/notifications')
}

export async function getUnreadCount() {
  return request<{ unread: number }>('/api/v1/notifications/unread')
}

export async function markNotificationRead(id: number) {
  return request<{ ok: boolean }>(`/api/v1/notifications/${id}/read`, { method: 'POST' })
}

export async function markAllNotificationsRead() {
  return request<{ ok: boolean }>('/api/v1/notifications/read-all', { method: 'POST' })
}

// ===== Tickets (Admin) =====

export async function adminGetTickets(status?: string) {
  const q = status && status !== 'all' ? `?status=${status}` : ''
  return adminRequest<{ items: any[] }>(`/api/admin/v1/tickets${q}`)
}

export async function adminGetTicketDetail(id: number) {
  return adminRequest<{ ticket: any; messages: any[] }>(`/api/admin/v1/tickets/${id}`)
}

export async function adminReplyTicket(id: number, content: string) {
  return adminRequest<{ ok: boolean }>(`/api/admin/v1/tickets/${id}/reply`, {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
}

export async function adminCloseTicket(id: number) {
  return adminRequest<{ ok: boolean }>(`/api/admin/v1/tickets/${id}/close`, {
    method: 'POST',
  })
}

export { ApiError }
