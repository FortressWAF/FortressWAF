export interface Site {
  id: string
  name: string
  domain: string
  originUrl: string
  status: 'online' | 'offline' | 'degraded'
  requestsToday: number
  attacksBlocked: number
  lastSeen: string
  createdAt: string
  techStack?: string
  rulesCount?: number
}

export interface Rule {
  id: string
  name: string
  description: string
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info'
  status: 'enabled' | 'disabled'
  category: string
  tags: string[]
  yaml: string
  createdAt: string
  updatedAt: string
  siteId?: string
  matchCount?: number
}

export interface LogEntry {
  id: string
  timestamp: string
  ip: string
  siteId: string
  siteName: string
  ruleId?: string
  ruleName?: string
  action: 'blocked' | 'allowed' | 'challenged' | 'logged'
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info'
  method: string
  path: string
  statusCode: number
  country: string
  userAgent: string
  requestHeaders?: Record<string, string>
  responseHeaders?: Record<string, string>
  requestBody?: string
  responseBody?: string
}

export interface TrafficPoint {
  timestamp: string
  value: number
  blocked?: number
  allowed?: number
}

export interface AttackSummary {
  totalRequests: number
  blockedAttacks: number
  activeThreats: number
  currentQps: number
  requestsTrend: number
  blockedTrend: number
  threatsTrend: number
  qpsTrend: number
}

export interface TopEndpoint {
  path: string
  requests: number
  attacks: number
  method: string
}

export interface AttackerIP {
  ip: string
  requests: number
  attacks: number
  country: string
  asn: string
  firstSeen: string
  lastSeen: string
}

export interface Alert {
  id: string
  type: 'attack' | 'anomaly' | 'system' | 'update'
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info'
  message: string
  timestamp: string
  siteName?: string
  read: boolean
}

export interface GeoData {
  country: string
  code: string
  attacks: number
  lat: number
  lng: number
}

export interface Patch {
  id: string
  cveId: string
  title: string
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info'
  status: 'draft' | 'testing' | 'deployed' | 'expired'
  affectedSites: string[]
  createdAt: string
  expiresAt: string
  description: string
  coverage: number
}

export interface NotificationConfig {
  slack?: { enabled: boolean; webhookUrl: string; channel: string }
  email?: { enabled: boolean; smtpHost: string; smtpPort: number; recipients: string[] }
  pagerduty?: { enabled: boolean; apiKey: string; serviceId: string }
  webhook?: { enabled: boolean; url: string; secret: string }
}

export interface ApiKey {
  id: string
  name: string
  key: string
  scopes: string[]
  createdAt: string
  expiresAt: string
  lastUsed: string
  status: 'active' | 'revoked'
}

export interface User {
  id: string
  name: string
  email: string
  avatar?: string
  role: 'admin' | 'editor' | 'viewer'
  mfaEnabled: boolean
}

export interface DashboardStats {
  summary: AttackSummary
  traffic: TrafficPoint[]
  topEndpoints: TopEndpoint[]
  attackers: AttackerIP[]
  alerts: Alert[]
  geoData: GeoData[]
}
