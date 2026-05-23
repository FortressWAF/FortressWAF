'use client'

import * as React from 'react'
import { Plus, Search, Upload, Play, MoreHorizontal } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription,
} from '@/components/ui/dialog'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useToast } from '@/components/ui/toast'
import { cn, formatDate, getSeverityColor } from '@/lib/utils'
import type { Rule } from '@/types'

const MOCK_RULES: Rule[] = [
  { id: '1', name: 'SQL Injection Detection', description: 'Detects common SQL injection patterns', severity: 'critical', status: 'enabled', category: 'owasp-top10', tags: ['injection', 'sqli'], yaml: 'name: sql-injection\nmatch:\n  - pattern: "(\'|\\")(\\s)*(or|and|union|select|insert|delete|update)(\\s)+"\n  - pattern: "1\\s*=\\s*1"\naction: block\nseverity: critical\n', createdAt: '2024-01-15', updatedAt: '2024-03-10', matchCount: 45200 },
  { id: '2', name: 'XSS Protection', description: 'Cross-site scripting attack prevention', severity: 'high', status: 'enabled', category: 'owasp-top10', tags: ['xss', 'injection'], yaml: 'name: xss-protection\nmatch:\n  - pattern: "<script[^>]*>"\n  - pattern: "on\\w+\\s*="\naction: block\nseverity: high\n', createdAt: '2024-01-15', updatedAt: '2024-03-08', matchCount: 28400 },
  { id: '3', name: 'Path Traversal', description: 'Prevents directory traversal attacks', severity: 'high', status: 'enabled', category: 'owasp-top10', tags: ['path', 'traversal'], yaml: 'name: path-traversal\nmatch:\n  - pattern: "\\.\\.\\/"\n  - pattern: "\\.\\.\\\\"\n  - pattern: "%2e%2e%2f"\naction: block\nseverity: high\n', createdAt: '2024-01-20', updatedAt: '2024-03-05', matchCount: 12500 },
  { id: '4', name: 'Rate Limiting Basic', description: '100 req/min per IP', severity: 'medium', status: 'enabled', category: 'rate-limit', tags: ['rate', 'dos'], yaml: 'name: rate-limit-basic\nmatch:\n  rate: 100\ntime_window: 60\naction: challenge\nseverity: medium\n', createdAt: '2024-02-01', updatedAt: '2024-03-01', matchCount: 89000 },
  { id: '5', name: 'Bot User-Agent Block', description: 'Blocks known malicious bots', severity: 'low', status: 'disabled', category: 'bot', tags: ['bot', 'user-agent'], yaml: 'name: bot-block\nmatch:\n  - pattern: "(nikto|masscan|acunetix|nessus|nmap|sqlmap|wpscan)"\naction: block\nseverity: low\n', createdAt: '2024-02-10', updatedAt: '2024-02-28', matchCount: 3400 },
  { id: '6', name: 'RCE Detection', description: 'Remote code execution attempt detection', severity: 'critical', status: 'enabled', category: 'owasp-top10', tags: ['rce', 'execution'], yaml: 'name: rce-detection\nmatch:\n  - pattern: "(system|exec|shell_exec|passthru|eval|assert|preg_replace)\\\s*\\("  \naction: block\nseverity: critical\n', createdAt: '2024-02-15', updatedAt: '2024-03-12', matchCount: 3200 },
  { id: '7', name: 'LFI Detection', description: 'Local file inclusion prevention', severity: 'high', status: 'enabled', category: 'owasp-top10', tags: ['lfi', 'file'], yaml: 'name: lfi-detection\nmatch:\n  - pattern: "(file|php|data|expect)://"\n  - pattern: "/etc/passwd"\naction: block\nseverity: high\n', createdAt: '2024-02-20', updatedAt: '2024-03-10', matchCount: 5600 },
  { id: '8', name: 'Comment Spam', description: 'Blocks comment spam patterns', severity: 'low', status: 'disabled', category: 'custom', tags: ['spam'], yaml: 'name: comment-spam\nmatch:\n  - pattern: "<a\\s+href[^>]*>\\s*buy\\s+"\naction: block\nseverity: low\n', createdAt: '2024-03-01', updatedAt: '2024-03-01', matchCount: 1200 },
]

export default function RulesPage() {
  const { toast } = useToast()
  const [search, setSearch] = React.useState('')
  const [severityFilter, setSeverityFilter] = React.useState('all')
  const [statusFilter, setStatusFilter] = React.useState('all')
  const [categoryFilter, setCategoryFilter] = React.useState('all')
  const [rules, setRules] = React.useState(MOCK_RULES)
  const [createOpen, setCreateOpen] = React.useState(false)
  const [testOpen, setTestOpen] = React.useState(false)
  const [testRuleId, setTestRuleId] = React.useState<string | null>(null)
  const [testResult, setTestResult] = React.useState<{ matched: boolean; ruleName?: string } | null>(null)
  const [yamlContent, setYamlContent] = React.useState(`name: custom-rule
description: Custom security rule
severity: medium
action: block
match:
  - pattern: "suspicious-pattern"`)

  const filtered = rules.filter((r) => {
    const matchesSearch = r.name.toLowerCase().includes(search.toLowerCase()) || r.tags.some((t) => t.includes(search.toLowerCase()))
    const matchesSeverity = severityFilter === 'all' || r.severity === severityFilter
    const matchesStatus = statusFilter === 'all' || r.status === statusFilter
    const matchesCategory = categoryFilter === 'all' || r.category === categoryFilter
    return matchesSearch && matchesSeverity && matchesStatus && matchesCategory
  })

  function toggleRule(id: string) {
    setRules((prev) =>
      prev.map((r) =>
        r.id === id ? { ...r, status: r.status === 'enabled' ? 'disabled' : 'enabled' as 'enabled' | 'disabled' } : r,
      ),
    )
    const rule = rules.find((r) => r.id === id)
    toast({ title: `Rule ${rule?.status === 'enabled' ? 'disabled' : 'enabled'}`, description: rule?.name })
  }

  function handleTestRule(ruleId: string) {
    setTestRuleId(ruleId)
    setTestResult(null)
    setTestOpen(true)
  }

  function runTest() {
    const matched = Math.random() > 0.5
    setTestResult({
      matched,
      ruleName: matched ? rules.find((r) => r.id === testRuleId)?.name : undefined,
    })
  }

  function deleteRule(id: string) {
    setRules((prev) => prev.filter((r) => r.id !== id))
    toast({ title: 'Rule deleted', variant: 'success' })
  }

  function handleImport() {
    toast({ title: 'Import started', description: 'Rules are being imported...' })
  }

  function handleCreateRule() {
    const yamlLines = yamlContent.split('\n')
    const nameLine = yamlLines.find((l) => l.startsWith('name:'))
    const name = nameLine ? nameLine.replace('name:', '').trim() : 'Custom Rule'
    setRules((prev) => [...prev, {
      id: String(Date.now()),
      name,
      description: 'Custom rule',
      severity: 'medium',
      status: 'enabled',
      category: 'custom',
      tags: [],
      yaml: yamlContent,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    }])
    setCreateOpen(false)
    toast({ title: 'Rule created', description: name, variant: 'success' })
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Rules</h1>
          <p className="text-muted-foreground">Manage WAF security rules</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleImport}>
            <Upload className="w-4 h-4 mr-2" /> Import
          </Button>
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="w-4 h-4 mr-2" /> Create Rule
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input placeholder="Search rules..." className="pl-9" value={search} onChange={(e) => setSearch(e.target.value)} />
            </div>
            <Select value={severityFilter} onValueChange={setSeverityFilter}>
              <SelectTrigger className="w-[130px]"><SelectValue placeholder="Severity" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Severities</SelectItem>
                <SelectItem value="critical">Critical</SelectItem>
                <SelectItem value="high">High</SelectItem>
                <SelectItem value="medium">Medium</SelectItem>
                <SelectItem value="low">Low</SelectItem>
              </SelectContent>
            </Select>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[130px]"><SelectValue placeholder="Status" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="enabled">Enabled</SelectItem>
                <SelectItem value="disabled">Disabled</SelectItem>
              </SelectContent>
            </Select>
            <Select value={categoryFilter} onValueChange={setCategoryFilter}>
              <SelectTrigger className="w-[140px]"><SelectValue placeholder="Category" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Categories</SelectItem>
                <SelectItem value="owasp-top10">OWASP Top 10</SelectItem>
                <SelectItem value="rate-limit">Rate Limit</SelectItem>
                <SelectItem value="bot">Bot</SelectItem>
                <SelectItem value="custom">Custom</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Severity</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Category</TableHead>
                <TableHead>Tags</TableHead>
                <TableHead className="text-right">Matches</TableHead>
                <TableHead>Updated</TableHead>
                <TableHead className="w-24">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((rule) => (
                <TableRow key={rule.id}>
                  <TableCell>
                    <div>
                      <p className="font-medium text-sm">{rule.name}</p>
                      <p className="text-xs text-muted-foreground">{rule.description}</p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className={getSeverityColor(rule.severity)}>
                      {rule.severity}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Switch
                      checked={rule.status === 'enabled'}
                      onCheckedChange={() => toggleRule(rule.id)}
                    />
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary" className="text-[10px]">{rule.category}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1 flex-wrap">
                      {rule.tags.map((tag) => (
                        <Badge key={tag} variant="outline" className="text-[10px]">{tag}</Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className="text-right font-mono text-xs">{rule.matchCount?.toLocaleString()}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{formatDate(rule.updatedAt)}</TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <MoreHorizontal className="w-4 h-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem>View details</DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleTestRule(rule.id)}>
                          <Play className="w-3 h-3 mr-2" /> Test rule
                        </DropdownMenuItem>
                        <DropdownMenuItem>Edit</DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem className="text-red-500" onClick={() => deleteRule(rule.id)}>Delete</DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-[640px]">
          <DialogHeader>
            <DialogTitle>Create Rule</DialogTitle>
            <DialogDescription>Define a new WAF rule using YAML</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <textarea
              className="w-full h-48 font-mono text-sm p-3 rounded-md border bg-background resize-none focus:outline-none focus:ring-2 focus:ring-ring"
              value={yamlContent}
              onChange={(e) => setYamlContent(e.target.value)}
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setCreateOpen(false)}>Cancel</Button>
              <Button onClick={handleCreateRule}>Create</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={testOpen} onOpenChange={setTestOpen}>
        <DialogContent className="sm:max-w-[480px]">
          <DialogHeader>
            <DialogTitle>Test Rule</DialogTitle>
            <DialogDescription>Paste a sample request to test against rule</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <textarea
              className="w-full h-32 font-mono text-sm p-3 rounded-md border bg-background resize-none focus:outline-none"
              placeholder="Paste request payload here..."
            />
            <div className="flex justify-between items-center">
              {testResult && (
                <div className={cn('text-sm font-medium', testResult.matched ? 'text-red-500' : 'text-green-500')}>
                  {testResult.matched ? '⚠ Pattern matched!' : '✓ No match found'}
                  {testResult.ruleName && <span className="text-muted-foreground ml-1">({testResult.ruleName})</span>}
                </div>
              )}
              <Button onClick={runTest} className="ml-auto">
                <Play className="w-4 h-4 mr-2" /> Run Test
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
