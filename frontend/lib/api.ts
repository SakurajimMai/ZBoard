const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://127.0.0.1:3000'

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

export async function adminGetPlans() {
  return adminRequest<{ items: any[] }>('/api/admin/v1/plans')
}

export async function adminCreatePlan(data: any) {
  return adminRequest<{ plan_id: number }>('/api/admin/v1/plans', {
    method: 'POST',
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

export { ApiError }
