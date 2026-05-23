'use client'

import * as React from 'react'
import { Activity, Download, ArrowUp, ArrowDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ChartContainer, AreaChart, BarChart, PieChart, Sparkline } from '@/components/ui/chart'
import { WorldMapHeatmap } from '@/components/ui/map'
import { useToast } from '@/components/ui/toast'
import { formatNumber, formatBytes } from '@/lib/utils'

const trafficData = Array.from({ length: 72 }, (_, i) => ({
  time: new Date(Date.now() - (71 - i) * 20 * 60000).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' }),
  requests: Math.floor(Math.random() * 1500) + 500,
  blocked: Math.floor(Math.random() * 300),
  baseline: Math.floor(Math.random() * 1200) + 400,
}))

const attackTrendDaily = Array.from({ length: 30 }, (_, i) => ({
  date: new Date(Date.now() - (29 - i) * 86400000).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
  attacks: Math.floor(Math.random() * 50000) + 10000,
}))

const responseTimeData = [
  { name: '< 100ms', value: 45, color: '#10b981' },
  { name: '100-200ms', value: 28, color: '#06b6d4' },
  { name: '200-500ms', value: 15, color: '#f59e0b' },
  { name: '500ms-1s', value: 8, color: '#f97316' },
  { name: '> 1s', value: 4, color: '#ef4444' },
]

const geoData = [
  { country: 'Russia', code: 'RU', attacks: 45230, lat: 61.524, lng: 105.318 },
  { country: 'China', code: 'CN', attacks: 32100, lat: 35.861, lng: 104.195 },
  { country: 'United States', code: 'US', attacks: 28400, lat: 37.090, lng: -95.712 },
  { country: 'Germany', code: 'DE', attacks: 12100, lat: 51.165, lng: 10.451 },
  { country: 'Netherlands', code: 'NL', attacks: 9800, lat: 52.132, lng: 5.291 },
  { country: 'Brazil', code: 'BR', attacks: 6800, lat: -14.235, lng: -51.925 },
  { country: 'India', code: 'IN', attacks: 5400, lat: 20.593, lng: 78.962 },
  { country: 'Singapore', code: 'SG', attacks: 3800, lat: 1.352, lng: 103.819 },
]

const anomalyTimeline = [
  { date: 'Mar 10', severity: 'critical', description: 'SQL injection spike — 12x baseline' },
  { date: 'Mar 14', severity: 'high', description: 'DDoS attempt from ASN AS9009' },
  { date: 'Mar 17', severity: 'medium', description: 'Scanning activity on /api/admin' },
  { date: 'Mar 19', severity: 'low', description: 'Rate limit threshold breached' },
]

export default function AnalyticsPage() {
  const { toast } = useToast()
  const [range, setRange] = React.useState('7d')

  function handleExport() {
    toast({ title: 'Analytics exported', description: 'Report downloaded as PDF', variant: 'success' })
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Analytics</h1>
          <p className="text-muted-foreground">Deep dive into traffic and attack patterns</p>
        </div>
        <div className="flex items-center gap-2">
          <Select value={range} onValueChange={setRange}>
            <SelectTrigger className="w-[130px]"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="24h">Last 24 Hours</SelectItem>
              <SelectItem value="7d">Last 7 Days</SelectItem>
              <SelectItem value="30d">Last 30 Days</SelectItem>
              <SelectItem value="90d">Last 90 Days</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" onClick={handleExport}>
            <Download className="w-4 h-4 mr-2" /> Export Report
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        {[
          { title: 'Avg Request/sec', value: formatNumber(1842), trend: 5.7, color: 'text-green-500' },
          { title: 'Bandwidth', value: formatBytes(2847521 * 2048), trend: 3.2, color: 'text-green-500' },
          { title: 'Avg Response Time', value: '142ms', trend: -8.1, color: 'text-green-500' },
          { title: 'Block Rate', value: '6.4%', trend: 2.1, color: 'text-red-500' },
        ].map((stat) => (
          <Card key={stat.title}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">{stat.title}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stat.value}</div>
              <div className={cn('flex items-center gap-1 text-xs mt-1', stat.color)}>
                {stat.trend >= 0 ? <ArrowUp className="w-3 h-3" /> : <ArrowDown className="w-3 h-3" />}
                {Math.abs(stat.trend)}% vs baseline
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Tabs defaultValue="traffic">
        <TabsList>
          <TabsTrigger value="traffic">Traffic Overview</TabsTrigger>
          <TabsTrigger value="attacks">Attack Trends</TabsTrigger>
          <TabsTrigger value="geo">Geo Distribution</TabsTrigger>
          <TabsTrigger value="endpoints">Top Endpoints</TabsTrigger>
          <TabsTrigger value="anomalies">Anomalies</TabsTrigger>
        </TabsList>

        <TabsContent value="traffic" className="space-y-6">
          <div className="grid gap-6 lg:grid-cols-2">
            <ChartContainer title="Request Volume" subtitle="Current vs baseline comparison">
              <AreaChart
                data={trafficData}
                xKey="time"
                series={[
                  { key: 'requests', color: '#06b6d4', name: 'Actual' },
                  { key: 'baseline', color: '#6b7280', name: 'Baseline' },
                ]}
              />
            </ChartContainer>
            <ChartContainer title="Response Time Distribution">
              <PieChart data={responseTimeData} />
            </ChartContainer>
          </div>
        </TabsContent>

        <TabsContent value="attacks" className="space-y-6">
          <ChartContainer title="Daily Attack Volume" subtitle="Last 30 days">
            <BarChart
              data={attackTrendDaily}
              xKey="date"
              series={[{ key: 'attacks', color: '#ef4444', name: 'Attacks' }]}
            />
          </ChartContainer>
        </TabsContent>

        <TabsContent value="geo" className="space-y-6">
          <div className="grid gap-6 lg:grid-cols-2">
            <WorldMapHeatmap data={geoData} title="Attack Origin Heat Map" />
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Attacks by Country</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Country</TableHead>
                      <TableHead className="text-right">Attacks</TableHead>
                      <TableHead className="text-right">Percentage</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {geoData.map((g) => (
                      <TableRow key={g.code}>
                        <TableCell>{g.country}</TableCell>
                        <TableCell className="text-right font-mono text-xs">{formatNumber(g.attacks)}</TableCell>
                        <TableCell className="text-right">{((g.attacks / geoData.reduce((a, b) => a + b.attacks, 0)) * 100).toFixed(1)}%</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="endpoints" className="space-y-6">
          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Top Endpoints</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Path</TableHead>
                      <TableHead>Method</TableHead>
                      <TableHead className="text-right">Requests</TableHead>
                      <TableHead className="text-right">Attacks</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {[
                      { path: '/api/v1/login', method: 'POST', requests: 38920, attacks: 8920 },
                      { path: '/api/v1/users', method: 'GET', requests: 28450, attacks: 1230 },
                      { path: '/api/v1/search', method: 'GET', requests: 22100, attacks: 2340 },
                      { path: '/api/v1/checkout', method: 'POST', requests: 18200, attacks: 890 },
                    ].map((ep) => (
                      <TableRow key={ep.path}>
                        <TableCell className="font-mono text-xs">{ep.path}</TableCell>
                        <TableCell><Badge variant="secondary" className="text-[10px] font-mono">{ep.method}</Badge></TableCell>
                        <TableCell className="text-right font-mono text-xs">{formatNumber(ep.requests)}</TableCell>
                        <TableCell className="text-right font-mono text-xs text-red-500">{formatNumber(ep.attacks)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Top User-Agents</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>User-Agent</TableHead>
                      <TableHead className="text-right">Count</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {[
                      { ua: 'Mozilla/5.0 (compatible; Googlebot/2.1)', count: 45200 },
                      { ua: 'python-requests/2.31.0', count: 28400 },
                      { ua: 'curl/8.4.0', count: 12300 },
                      { ua: 'Mozilla/5.0 (compatible; SemrushBot)', count: 8900 },
                    ].map((ua) => (
                      <TableRow key={ua.ua}>
                        <TableCell className="font-mono text-xs max-w-[300px] truncate">{ua.ua}</TableCell>
                        <TableCell className="text-right font-mono text-xs">{formatNumber(ua.count)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="anomalies" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Anomaly Timeline</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <div className="absolute left-4 top-0 bottom-0 w-px bg-border" />
                <div className="space-y-6">
                  {anomalyTimeline.map((anomaly, i) => (
                    <div key={i} className="relative flex items-start gap-4 pl-10">
                      <div className={cn(
                        'absolute left-2.5 w-3 h-3 rounded-full border-2 bg-background',
                        anomaly.severity === 'critical' ? 'border-red-500' :
                        anomaly.severity === 'high' ? 'border-orange-500' :
                        anomaly.severity === 'medium' ? 'border-yellow-500' : 'border-blue-500',
                      )} />
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium">{anomaly.date}</span>
                          <Badge variant="outline" className={cn(
                            'text-[10px]',
                            anomaly.severity === 'critical' ? 'text-red-500 border-red-500' :
                            anomaly.severity === 'high' ? 'text-orange-500 border-orange-500' :
                            anomaly.severity === 'medium' ? 'text-yellow-500 border-yellow-500' : 'text-blue-500 border-blue-500',
                          )}>{anomaly.severity}</Badge>
                        </div>
                        <p className="text-sm text-muted-foreground mt-1">{anomaly.description}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

function cn(...classes: (string | boolean | undefined | null)[]) {
  return classes.filter(Boolean).join(' ')
}
