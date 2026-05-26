'use client'

import * as React from 'react'
import { ArrowUp, ArrowDown, Shield, Activity, AlertTriangle, Zap } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ChartContainer, Sparkline, AreaChart, PieChart } from '@/components/ui/chart'
import { WorldMapHeatmap } from '@/components/ui/map'
import { Skeleton } from '@/components/ui/skeleton'
import { formatNumber, formatDate, cn } from '@/lib/utils'
import type { DashboardStats, TrafficPoint, TopEndpoint, AttackerIP, Alert } from '@/types'

const MOCK_STATS: DashboardStats = {
  summary: {
    totalRequests: 2847521,
    blockedAttacks: 182456,
    activeThreats: 23,
    currentQps: 1842,
    requestsTrend: 12.5,
    blockedTrend: -3.2,
    threatsTrend: 8.1,
    qpsTrend: 5.7,
  },
  traffic: Array.from({ length: 288 }, (_, i) => ({
    timestamp: new Date(Date.now() - (287 - i) * 5 * 60000).toISOString(),
    value: Math.floor(Math.random() * 2000) + 500,
    blocked: Math.floor(Math.random() * 200),
    allowed: Math.floor(Math.random() * 1800) + 300,
  })),
  topEndpoints: [
    { path: '/api/v1/users', requests: 45210, attacks: 1230, method: 'POST' },
    { path: '/api/v1/login', requests: 38920, attacks: 8920, method: 'POST' },
    { path: '/api/v1/products', requests: 28450, attacks: 450, method: 'GET' },
    { path: '/api/v1/search', requests: 22100, attacks: 2340, method: 'GET' },
    { path: '/api/v1/checkout', requests: 18200, attacks: 890, method: 'POST' },
    { path: '/api/v1/admin', requests: 1200, attacks: 980, method: 'GET' },
    { path: '/wp-admin', requests: 890, attacks: 840, method: 'GET' },
    { path: '/.env', requests: 560, attacks: 520, method: 'GET' },
  ],
  attackers: [
    { ip: '185.220.101.42', requests: 45210, attacks: 12340, country: 'RU', asn: 'AS9009', firstSeen: '2024-01-15', lastSeen: '2024-03-20' },
    { ip: '103.235.46.91', requests: 32100, attacks: 8920, country: 'CN', asn: 'AS4837', firstSeen: '2024-02-01', lastSeen: '2024-03-20' },
    { ip: '45.33.32.156', requests: 28400, attacks: 7650, country: 'US', asn: 'AS6939', firstSeen: '2024-01-20', lastSeen: '2024-03-20' },
    { ip: '91.121.87.34', requests: 19800, attacks: 5430, country: 'FR', asn: 'AS16276', firstSeen: '2024-02-10', lastSeen: '2024-03-20' },
    { ip: '5.188.62.18', requests: 15400, attacks: 4320, country: 'NL', asn: 'AS202306', firstSeen: '2024-03-01', lastSeen: '2024-03-20' },
  ],
  alerts: [
    { id: '1', type: 'attack', severity: 'critical', message: 'SQL injection attempt detected on /api/v1/login', timestamp: new Date().toISOString(), siteName: 'Main API', read: false },
    { id: '2', type: 'attack', severity: 'high', message: 'XSS payload detected in search parameter', timestamp: new Date(Date.now() - 300000).toISOString(), siteName: 'E-commerce', read: false },
    { id: '3', type: 'anomaly', severity: 'medium', message: 'Unusual traffic spike from ASN AS9009', timestamp: new Date(Date.now() - 600000).toISOString(), read: false },
    { id: '4', type: 'system', severity: 'low', message: 'Rate limit threshold reached for /api/search', timestamp: new Date(Date.now() - 3600000).toISOString(), read: false },
    { id: '5', type: 'update', severity: 'info', message: 'Rule set updated: CVE-2024-1234 patch deployed', timestamp: new Date(Date.now() - 7200000).toISOString(), read: true },
  ],
  geoData: [
    { country: 'Russia', code: 'RU', attacks: 45230, lat: 61.524, lng: 105.318 },
    { country: 'China', code: 'CN', attacks: 32100, lat: 35.861, lng: 104.195 },
    { country: 'United States', code: 'US', attacks: 28400, lat: 37.090, lng: -95.712 },
    { country: 'Germany', code: 'DE', attacks: 12100, lat: 51.165, lng: 10.451 },
    { country: 'Netherlands', code: 'NL', attacks: 9800, lat: 52.132, lng: 5.291 },
    { country: 'France', code: 'FR', attacks: 7600, lat: 46.603, lng: 1.888 },
    { country: 'Brazil', code: 'BR', attacks: 6800, lat: -14.235, lng: -51.925 },
    { country: 'India', code: 'IN', attacks: 5400, lat: 20.593, lng: 78.962 },
    { country: 'United Kingdom', code: 'GB', attacks: 4200, lat: 55.378, lng: -3.435 },
    { country: 'Singapore', code: 'SG', attacks: 3800, lat: 1.352, lng: 103.819 },
  ],
}

function StatCard({
  title,
  value,
  trend,
  icon,
  sparklineData,
}: {
  title: string
  value: string
  trend: number
  icon: React.ReactNode
  sparklineData: { value: number }[]
}) {
  const isUp = trend >= 0
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-xs">{title}</CardTitle>
        <div className="p-2 border-2 border-foreground bg-primary/10 text-primary">{icon}</div>
      </CardHeader>
      <CardContent>
        <div className="flex items-baseline justify-between">
          <div>
            <div className="text-3xl font-black">{value}</div>
            <div className={cn('flex items-center gap-1 text-xs font-bold mt-1', isUp ? 'text-green-600' : 'text-red-600')}>
              {isUp ? <ArrowUp className="w-3 h-3" /> : <ArrowDown className="w-3 h-3" />}
              <span>{Math.abs(trend)}% vs last hour</span>
            </div>
          </div>
          <div className="w-24">
            <Sparkline data={sparklineData} color={isUp ? '#10b981' : '#ef4444'} />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function AlertItem({ alert }: { alert: Alert }) {
  const severityColors: Record<string, string> = {
    critical: 'border-foreground bg-red-100 dark:bg-red-950',
    high: 'border-foreground bg-orange-100 dark:bg-orange-950',
    medium: 'border-foreground bg-yellow-100 dark:bg-yellow-950',
    low: 'border-foreground bg-blue-100 dark:bg-blue-950',
    info: 'border-foreground bg-green-100 dark:bg-green-950',
  }
  return (
    <div className={cn('flex items-start gap-3 p-3 border-2 border-foreground', severityColors[alert.severity])}>
      <div className={cn('w-3 h-3 mt-1.5 shrink-0 border-2 border-foreground', alert.read ? 'bg-muted-foreground' : 'bg-primary')} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-bold truncate">{alert.message}</p>
        <div className="flex items-center gap-2 mt-1">
          <Badge variant="outline" className="text-[10px] px-1.5 py-0 font-bold">{alert.severity}</Badge>
          {alert.siteName && <span className="text-xs text-muted-foreground font-medium">{alert.siteName}</span>}
          <span className="text-xs text-muted-foreground font-medium">{formatDate(alert.timestamp)}</span>
        </div>
      </div>
    </div>
  )
}

let idCounter = 0
function getSparklineData(): { value: number }[] {
  const id = idCounter++
  const seed = (id * 7 + 13) % 37
  return Array.from({ length: 24 }, (_, i) => ({
    value: Math.floor(Math.sin((i + seed) * 0.5) * 100 + 200 + Math.random() * 100),
  }))
}

export default function DashboardPage() {
  const [loading, setLoading] = React.useState(false)
  const stats = MOCK_STATS
  const trafficData = React.useMemo(() => {
    return stats.traffic.map((t: TrafficPoint) => ({
      time: new Date(t.timestamp).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' }),
      requests: t.value,
      blocked: t.blocked || 0,
    }))
  }, [stats.traffic])

  const attackTypeData = [
    { name: 'SQL Injection', value: 35, color: '#ef4444' },
    { name: 'XSS', value: 25, color: '#f59e0b' },
    { name: 'Path Traversal', value: 15, color: '#8b5cf6' },
    { name: 'RCE', value: 10, color: '#ec4899' },
    { name: 'LFI', value: 8, color: '#14b8a6' },
    { name: 'Other', value: 7, color: '#6b7280' },
  ]

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i}><CardContent className="p-5"><Skeleton className="h-24" /></CardContent></Card>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between border-b-2 border-foreground pb-4">
        <div>
          <h1 className="text-2xl font-black uppercase tracking-tight">Overview</h1>
          <p className="text-muted-foreground font-medium">Real-time WAF security dashboard</p>
        </div>
        <Button variant="outline" onClick={() => setLoading(true)} className="hidden sm:flex">
          <Activity className="w-4 h-4 mr-2" /> Refresh
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard title="Total Requests" value={formatNumber(stats.summary.totalRequests)} trend={stats.summary.requestsTrend} icon={<Activity className="w-4 h-4" />} sparklineData={getSparklineData()} />
        <StatCard title="Blocked Attacks" value={formatNumber(stats.summary.blockedAttacks)} trend={stats.summary.blockedTrend} icon={<Shield className="w-4 h-4" />} sparklineData={getSparklineData()} />
        <StatCard title="Active Threats" value={String(stats.summary.activeThreats)} trend={stats.summary.threatsTrend} icon={<AlertTriangle className="w-4 h-4" />} sparklineData={getSparklineData()} />
        <StatCard title="Current QPS" value={formatNumber(stats.summary.currentQps)} trend={stats.summary.qpsTrend} icon={<Zap className="w-4 h-4" />} sparklineData={getSparklineData()} />
      </div>

      <div className="grid gap-6 lg:grid-cols-7">
        <div className="lg:col-span-5">
          <ChartContainer title="Traffic Overview" subtitle="Requests per second over the last 24 hours">
            <AreaChart
              data={trafficData}
              xKey="time"
              series={[
                { key: 'requests', color: '#06b6d4', name: 'Requests' },
                { key: 'blocked', color: '#ef4444', name: 'Blocked' },
              ]}
            />
          </ChartContainer>
        </div>
        <div className="lg:col-span-2">
          <ChartContainer title="Attack Distribution" subtitle="By attack type">
            <PieChart data={attackTypeData} />
          </ChartContainer>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-7">
        <div className="lg:col-span-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-xs">Top Attacked Endpoints</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Path</TableHead>
                    <TableHead>Method</TableHead>
                    <TableHead className="text-right">Requests</TableHead>
                    <TableHead className="text-right">Attacks</TableHead>
                    <TableHead className="text-right">Block Rate</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {stats.topEndpoints.map((ep: TopEndpoint) => (
                    <TableRow key={ep.path}>
                      <TableCell className="font-mono text-xs max-w-[200px] truncate">{ep.path}</TableCell>
                      <TableCell>
                        <Badge variant="secondary" className="font-mono text-[10px]">{ep.method}</Badge>
                      </TableCell>
                      <TableCell className="text-right font-bold">{formatNumber(ep.requests)}</TableCell>
                      <TableCell className="text-right text-destructive font-bold">{formatNumber(ep.attacks)}</TableCell>
                      <TableCell className="text-right">
                        <Badge variant={ep.attacks / ep.requests > 0.1 ? 'destructive' : 'secondary'}>
                          {((ep.attacks / ep.requests) * 100).toFixed(1)}%
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>
        <div className="lg:col-span-3 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-xs">Recent Alerts</CardTitle>
            </CardHeader>
            <CardContent className="p-3 space-y-2 max-h-[360px] overflow-y-auto scrollbar-thin">
              {stats.alerts.map((alert: Alert) => (
                <AlertItem key={alert.id} alert={alert} />
              ))}
            </CardContent>
          </Card>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-7">
        <div className="lg:col-span-4">
          <WorldMapHeatmap data={stats.geoData} title="Attack Origins" />
        </div>
        <div className="lg:col-span-3">
          <Card>
            <CardHeader>
              <CardTitle className="text-xs">Top Attacker IPs</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>IP</TableHead>
                    <TableHead>Country</TableHead>
                    <TableHead className="text-right">Attacks</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {stats.attackers.map((a: AttackerIP) => (
                    <TableRow key={a.ip}>
                      <TableCell className="font-mono text-xs font-bold">{a.ip}</TableCell>
                      <TableCell className="font-bold">{a.country}</TableCell>
                      <TableCell className="text-right text-destructive font-bold">{formatNumber(a.attacks)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
