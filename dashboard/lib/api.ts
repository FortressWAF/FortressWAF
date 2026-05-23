import type {
  Site, Rule, LogEntry, TrafficPoint, AttackSummary, TopEndpoint,
  AttackerIP, Alert, GeoData, Patch, NotificationConfig, ApiKey,
  User, DashboardStats,
} from '@/types'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'

let authToken: string | null = null

if (typeof window !== 'undefined') {
  authToken = localStorage.getItem('fortresswaf_token')
}

export function setToken(token: string | null) {
  authToken = token
  if (typeof window !== 'undefined') {
    if (token) localStorage.setItem('fortresswaf_token', token)
    else localStorage.removeItem('fortresswaf_token')
  }
}

export function getToken(): string | null {
  return authToken
}

class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public code?: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }

  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  })

  if (response.status === 429) {
    const retryAfter = response.headers.get('Retry-After')
    throw new ApiError(429, `Rate limited. Retry after ${retryAfter}s`, 'RATE_LIMITED')
  }

  if (response.status === 401) {
    setToken(null)
    if (typeof window !== 'undefined') {
      window.location.href = '/'
    }
    throw new ApiError(401, 'Unauthorized', 'UNAUTHORIZED')
  }

  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new ApiError(response.status, body.message || response.statusText, body.code)
  }

  if (response.status === 204) return undefined as T
  return response.json()
}

export const api = {
  auth: {
    login: (email: string, password: string) =>
      request<{ token: string; user: User }>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      }),
    sso: (provider: string) =>
      request<{ token: string; user: User }>('/auth/sso', {
        method: 'POST',
        body: JSON.stringify({ provider }),
      }),
    me: () => request<User>('/auth/me'),
  },

  sites: {
    list: (params?: Record<string, string>) =>
      request<Site[]>(`/sites?${new URLSearchParams(params)}`),
    get: (id: string) => request<Site>(`/sites/${id}`),
    create: (data: Partial<Site>) =>
      request<Site>('/sites', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: Partial<Site>) =>
      request<Site>(`/sites/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request<void>(`/sites/${id}`, { method: 'DELETE' }),
    detect: (originUrl: string) =>
      request<{ framework: string; version: string }>('/sites/detect', {
        method: 'POST',
        body: JSON.stringify({ originUrl }),
      }),
    traffic: (id: string, range?: string) =>
      request<TrafficPoint[]>(`/sites/${id}/traffic?range=${range || '24h'}`),
  },

  rules: {
    list: (params?: Record<string, string>) =>
      request<Rule[]>(`/rules?${new URLSearchParams(params)}`),
    get: (id: string) => request<Rule>(`/rules/${id}`),
    create: (data: Partial<Rule>) =>
      request<Rule>('/rules', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: Partial<Rule>) =>
      request<Rule>(`/rules/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request<void>(`/rules/${id}`, { method: 'DELETE' }),
    toggle: (id: string, enabled: boolean) =>
      request<Rule>(`/rules/${id}/toggle`, {
        method: 'POST',
        body: JSON.stringify({ enabled }),
      }),
    test: (id: string, requestData: Record<string, unknown>) =>
      request<{ matched: boolean; ruleName?: string }>(`/rules/${id}/test`, {
        method: 'POST',
        body: JSON.stringify(requestData),
      }),
    import: (format: string, data: string) =>
      request<{ imported: number; errors: string[] }>('/rules/import', {
        method: 'POST',
        body: JSON.stringify({ format, data }),
      }),
    templates: (framework: string) =>
      request<Rule[]>(`/rules/templates?framework=${framework}`),
  },

  logs: {
    list: (params?: Record<string, string>) =>
      request<LogEntry[]>(`/logs?${new URLSearchParams(params)}`),
    get: (id: string) => request<LogEntry>(`/logs/${id}`),
    export: (format: 'csv' | 'json', params?: Record<string, string>) =>
      request<string>(`/logs/export/${format}?${new URLSearchParams(params)}`),
    stats: (params?: Record<string, string>) =>
      request<{ total: number; blocked: number; uniqueIps: number }>(
        `/logs/stats?${new URLSearchParams(params)}`,
      ),
  },

  analytics: {
    dashboard: (range?: string) =>
      request<DashboardStats>(`/analytics/dashboard?range=${range || '24h'}`),
    traffic: (range?: string, granularity?: string) =>
      request<TrafficPoint[]>(`/analytics/traffic?range=${range || '24h'}&granularity=${granularity || '5m'}`),
    geo: (range?: string) =>
      request<GeoData[]>(`/analytics/geo?range=${range || '24h'}`),
    attackers: (range?: string, limit?: number) =>
      request<AttackerIP[]>(`/analytics/attackers?range=${range || '24h'}&limit=${limit || 10}`),
    endpoints: (range?: string, limit?: number) =>
      request<TopEndpoint[]>(`/analytics/endpoints?range=${range || '24h'}&limit=${limit || 10}`),
    baseline: (range?: string) =>
      request<{ current: TrafficPoint[]; baseline: TrafficPoint[] }>(
        `/analytics/baseline?range=${range || '24h'}`,
      ),
  },

  patches: {
    list: (params?: Record<string, string>) =>
      request<Patch[]>(`/patches?${new URLSearchParams(params)}`),
    get: (id: string) => request<Patch>(`/patches/${id}`),
    create: (data: Partial<Patch>) =>
      request<Patch>('/patches', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: Partial<Patch>) =>
      request<Patch>(`/patches/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request<void>(`/patches/${id}`, { method: 'DELETE' }),
    deploy: (id: string) =>
      request<Patch>(`/patches/${id}/deploy`, { method: 'POST' }),
    coverage: () =>
      request<{ total: number; covered: number; percentage: number }>('/patches/coverage'),
    fromCve: (cveId: string) =>
      request<{ patchId: string }>('/patches/from-cve', {
        method: 'POST',
        body: JSON.stringify({ cveId }),
      }),
  },

  settings: {
    get: () => request<Record<string, unknown>>('/settings'),
    update: (data: Record<string, unknown>) =>
      request<Record<string, unknown>>('/settings', {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    notifications: {
      get: () => request<NotificationConfig>('/settings/notifications'),
      update: (data: NotificationConfig) =>
        request<NotificationConfig>('/settings/notifications', {
          method: 'PUT',
          body: JSON.stringify(data),
        }),
      test: (type: string) =>
        request<{ sent: boolean }>(`/settings/notifications/test/${type}`, {
          method: 'POST',
        }),
    },
    tls: {
      upload: (cert: string, key: string) =>
        request<{ expiresAt: string }>('/settings/tls', {
          method: 'POST',
          body: JSON.stringify({ cert, key }),
        }),
      letsencrypt: (domain: string) =>
        request<{ status: string }>('/settings/tls/letsencrypt', {
          method: 'POST',
          body: JSON.stringify({ domain }),
        }),
    },
    apiKeys: {
      list: () => request<ApiKey[]>('/settings/api-keys'),
      create: (data: { name: string; scopes: string[] }) =>
        request<ApiKey>('/settings/api-keys', {
          method: 'POST',
          body: JSON.stringify(data),
        }),
      revoke: (id: string) =>
        request<void>(`/settings/api-keys/${id}/revoke`, { method: 'POST' }),
    },
  },
}
