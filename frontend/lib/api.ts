// API base URL.
// 默认走同源(空字符串)+ Next.js rewrites 反代到后端,这样:
//   - 走 CDN 时只需代理 80/443,不必暴露 :3000
//   - 浏览器无跨域,无需 CORS 预检
// 如果显式指定 NEXT_PUBLIC_API_URL 则走绝对地址(用于独立 API 子域名场景)。
function getApiBase(): string {
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL
  }
  return ''
}

const API_BASE = getApiBase()

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

export async function sendEmailCode(email: string, purpose: 'register' | 'reset_password') {
  return request<{ ok: boolean }>('/api/v1/auth/send-email-code', {
    method: 'POST',
    body: JSON.stringify({ email, purpose }),
  })
}

export async function registerWithCode(email: string, password: string, code: string) {
  return request<{ user_id: number }>('/api/v1/auth/register-with-code', {
    method: 'POST',
    body: JSON.stringify({ email, password, code }),
  })
}

export async function resetPassword(email: string, newPassword: string, code: string) {
  return request<{ ok: boolean }>('/api/v1/auth/reset-password', {
    method: 'POST',
    body: JSON.stringify({ email, new_password: newPassword, code }),
  })
}

export async function register(email: string, password: string) {
  return request<{ user_id: number }>('/api/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

export async function login(email: string, password: string) {
  const data = await request<{ token: string; user: any }>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
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

export async function getPlans() {
  return request<{ items: any[] }>('/api/v1/plans')
}

export async function createOrder(planId: number) {
  return request<{ order: any; existing: boolean }>('/api/v1/orders', {
    method: 'POST',
    body: JSON.stringify({ plan_id: planId }),
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

export async function adminGetUsers() {
  return adminRequest<{ items: any[] }>('/api/admin/v1/users')
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

export async function adminGetOrders() {
  return adminRequest<{ items: any[] }>('/api/admin/v1/orders')
}

export async function adminGetNodes() {
  return adminRequest<{ items: any[] }>('/api/admin/v1/nodes')
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

export async function adminGetPlans() {
  return adminRequest<{ items: any[] }>('/api/admin/v1/plans')
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

// ===== Tickets (User) =====

export async function getTickets() {
  return request<{ items: any[] }>('/api/v1/tickets')
}

export async function createTicket(subject: string, category: string, content: string) {
  return request<{ ticket_id: number; ticket_no: string }>('/api/v1/tickets', {
    method: 'POST',
    body: JSON.stringify({ subject, category, content }),
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
