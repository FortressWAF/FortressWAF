'use client'

import * as React from 'react'
import { Plus, Download, Trash2, Search, MoreHorizontal } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useToast } from '@/components/ui/toast'
import { SiteWizard } from './wizard'
import { cn, formatDate, formatNumber, getStatusDot } from '@/lib/utils'
import type { Site } from '@/types'

const MOCK_SITES: Site[] = [
  { id: '1', name: 'Main API', domain: 'api.example.com', originUrl: 'https://api.internal:8080', status: 'online', requestsToday: 1452000, attacksBlocked: 82300, lastSeen: new Date().toISOString(), createdAt: '2024-01-15', techStack: 'Node.js/Express', rulesCount: 45 },
  { id: '2', name: 'E-commerce', domain: 'shop.example.com', originUrl: 'https://shop.internal:3000', status: 'online', requestsToday: 892000, attacksBlocked: 45200, lastSeen: new Date().toISOString(), createdAt: '2024-01-20', techStack: 'Next.js', rulesCount: 38 },
  { id: '3', name: 'Admin Panel', domain: 'admin.example.com', originUrl: 'https://admin.internal:3001', status: 'degraded', requestsToday: 234000, attacksBlocked: 18900, lastSeen: new Date(Date.now() - 600000).toISOString(), createdAt: '2024-02-01', techStack: 'React/Vite', rulesCount: 52 },
  { id: '4', name: 'Legacy App', domain: 'legacy.example.com', originUrl: 'https://legacy.internal:8080', status: 'offline', requestsToday: 0, attacksBlocked: 0, lastSeen: new Date(Date.now() - 86400000).toISOString(), createdAt: '2024-02-10', techStack: 'PHP/Laravel', rulesCount: 28 },
  { id: '5', name: 'Docs Portal', domain: 'docs.example.com', originUrl: 'https://docs.internal:3000', status: 'online', requestsToday: 345000, attacksBlocked: 12000, lastSeen: new Date().toISOString(), createdAt: '2024-03-01', techStack: 'Docusaurus', rulesCount: 15 },
]

export default function SitesPage() {
  const { toast } = useToast()
  const [search, setSearch] = React.useState('')
  const [wizardOpen, setWizardOpen] = React.useState(false)
  const [selectedSites, setSelectedSites] = React.useState<string[]>([])
  const [sites, setSites] = React.useState(MOCK_SITES)

  const filtered = sites.filter((s) =>
    s.name.toLowerCase().includes(search.toLowerCase()) ||
    s.domain.toLowerCase().includes(search.toLowerCase()),
  )

  function toggleSelect(id: string) {
    setSelectedSites((prev) =>
      prev.includes(id) ? prev.filter((s) => s !== id) : [...prev, id],
    )
  }

  function handleBulkAction(action: string) {
    if (selectedSites.length === 0) {
      toast({ title: 'No sites selected', description: 'Please select sites first', variant: 'destructive' })
      return
    }
    if (action === 'delete') {
      setSites((prev) => prev.filter((s) => !selectedSites.includes(s.id)))
      toast({ title: 'Sites deleted', description: `${selectedSites.length} sites removed`, variant: 'success' })
      setSelectedSites([])
    } else if (action === 'enable') {
      toast({ title: 'Sites enabled', description: `${selectedSites.length} sites enabled`, variant: 'success' })
    } else if (action === 'disable') {
      toast({ title: 'Sites disabled', description: `${selectedSites.length} sites disabled` })
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Sites</h1>
          <p className="text-muted-foreground">Manage your protected web applications</p>
        </div>
        <div className="flex items-center gap-2">
          {selectedSites.length > 0 && (
            <>
              <Button variant="outline" size="sm" onClick={() => handleBulkAction('enable')}>Enable</Button>
              <Button variant="outline" size="sm" onClick={() => handleBulkAction('disable')}>Disable</Button>
              <Button variant="outline" size="sm" onClick={() => handleBulkAction('export')}>
                <Download className="w-4 h-4 mr-1" /> Export
              </Button>
              <Button variant="destructive" size="sm" onClick={() => handleBulkAction('delete')}>
                <Trash2 className="w-4 h-4 mr-1" /> Delete
              </Button>
            </>
          )}
          <Button onClick={() => setWizardOpen(true)}>
            <Plus className="w-4 h-4 mr-2" /> Add Site
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                placeholder="Search sites..."
                className="pl-9"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <Badge variant="secondary">{sites.length} total</Badge>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">
                  <input
                    type="checkbox"
                    className="rounded border-input"
                    checked={selectedSites.length === filtered.length && filtered.length > 0}
                    onChange={() => {
                      if (selectedSites.length === filtered.length) setSelectedSites([])
                      else setSelectedSites(filtered.map((s) => s.id))
                    }}
                  />
                </TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Domain</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Requests Today</TableHead>
                <TableHead className="text-right">Attacks Blocked</TableHead>
                <TableHead>Last Seen</TableHead>
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((site) => (
                <TableRow key={site.id}>
                  <TableCell>
                    <input
                      type="checkbox"
                      className="rounded border-input"
                      checked={selectedSites.includes(site.id)}
                      onChange={() => toggleSelect(site.id)}
                    />
                  </TableCell>
                  <TableCell className="font-medium">{site.name}</TableCell>
                  <TableCell className="font-mono text-xs">{site.domain}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span className={cn('w-2 h-2 rounded-full', getStatusDot(site.status))} />
                      <span className="capitalize text-sm">{site.status}</span>
                    </div>
                  </TableCell>
                  <TableCell className="text-right font-mono text-xs">{formatNumber(site.requestsToday)}</TableCell>
                  <TableCell className="text-right font-mono text-xs text-red-500">{formatNumber(site.attacksBlocked)}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{formatDate(site.lastSeen)}</TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <MoreHorizontal className="w-4 h-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem>View details</DropdownMenuItem>
                        <DropdownMenuItem>Edit site</DropdownMenuItem>
                        <DropdownMenuItem>View rules</DropdownMenuItem>
                        <DropdownMenuItem>View logs</DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem className="text-red-500">Delete</DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
              {filtered.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                    No sites found
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <SiteWizard open={wizardOpen} onOpenChange={setWizardOpen} />
    </div>
  )
}
