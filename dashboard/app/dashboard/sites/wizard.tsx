'use client'

import * as React from 'react'
import { Check, ChevronLeft, ChevronRight, Globe, Shield, Code, Server, Zap } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { useToast } from '@/components/ui/toast'
import { cn } from '@/lib/utils'

const STEPS = [
  { title: 'Origin', icon: Server },
  { title: 'Detect', icon: Code },
  { title: 'Rules', icon: Shield },
  { title: 'DNS', icon: Globe },
  { title: 'Summary', icon: Zap },
]

const DETECTED_STACKS = [
  { name: 'Node.js/Express', version: '18.x', confidence: 98 },
  { name: 'React', version: '18.2', confidence: 95 },
  { name: 'Next.js', version: '14.1', confidence: 92 },
]

const RULE_TEMPLATES = [
  { id: 'owasp-top10', name: 'OWASP Top 10', description: 'Core security rules for OWASP Top 10 vulnerabilities', selected: true },
  { id: 'sql-injection', name: 'SQL Injection', description: 'Advanced SQL injection detection patterns', selected: true },
  { id: 'xss', name: 'Cross-Site Scripting', description: 'XSS attack prevention rules', selected: true },
  { id: 'csrf', name: 'CSRF Protection', description: 'Cross-site request forgery defenses', selected: false },
  { id: 'rate-limit', name: 'Rate Limiting', description: 'Request rate limiting per IP', selected: true },
  { id: 'bot-block', name: 'Bot Blocking', description: 'Known bot and crawler blocking', selected: false },
  { id: 'geo-block', name: 'Geo-Blocking', description: 'Country-based access control', selected: false },
  { id: 'cors', name: 'CORS Headers', description: 'Cross-origin resource sharing policies', selected: true },
]

interface SiteWizardProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SiteWizard({ open, onOpenChange }: SiteWizardProps) {
  const { toast } = useToast()
  const [step, setStep] = React.useState(0)
  const [originUrl, setOriginUrl] = React.useState('')
  const [originPort, setOriginPort] = React.useState('8080')
  const [detectedStack, setDetectedStack] = React.useState<string | null>(null)
  const [detecting, setDetecting] = React.useState(false)
  const [templates, setTemplates] = React.useState(RULE_TEMPLATES)
  const [siteName, setSiteName] = React.useState('')

  function handleDetect() {
    if (!originUrl) {
      toast({ title: 'Enter an origin URL', variant: 'destructive' })
      return
    }
    setDetecting(true)
    setTimeout(() => {
      setDetectedStack('Node.js/Express')
      setDetecting(false)
      toast({ title: 'Framework detected', description: 'Node.js/Express v18.x detected', variant: 'success' })
    }, 1500)
  }

  function toggleTemplate(id: string) {
    setTemplates((prev) =>
      prev.map((t) => (t.id === id ? { ...t, selected: !t.selected } : t)),
    )
  }

  function handleFinish() {
    toast({
      title: 'Site deployed!',
      description: `${siteName || originUrl} is now protected by FortressWAF`,
      variant: 'success',
    })
    onOpenChange(false)
    setStep(0)
    setOriginUrl('')
    setOriginPort('8080')
    setDetectedStack(null)
    setSiteName('')
  }

  function handleClose() {
    onOpenChange(false)
    setStep(0)
    setOriginUrl('')
    setOriginPort('8080')
    setDetectedStack(null)
    setSiteName('')
  }

  const canProceed = () => {
    switch (step) {
      case 0: return originUrl.length > 0
      case 1: return true
      case 2: return templates.some((t) => t.selected)
      case 3: return true
      case 4: return true
      default: return true
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>Add New Site</DialogTitle>
          <DialogDescription>Protect your web application with FortressWAF</DialogDescription>
        </DialogHeader>

        <div className="flex justify-between mb-8">
          {STEPS.map((s, i) => {
            const Icon = s.icon
            const isCompleted = i < step
            const isCurrent = i === step
            return (
              <div key={s.title} className="flex flex-col items-center gap-1.5">
                <div
                  className={cn(
                    'flex items-center justify-center w-10 h-10 rounded-full border-2 transition-colors',
                    isCompleted && 'bg-fortress-600 border-fortress-600 text-white',
                    isCurrent && 'border-fortress-500 text-fortress-400',
                    !isCompleted && !isCurrent && 'border-muted text-muted-foreground',
                  )}
                >
                  {isCompleted ? <Check className="w-4 h-4" /> : <Icon className="w-4 h-4" />}
                </div>
                <span className={cn('text-xs', isCurrent ? 'text-fortress-400 font-medium' : 'text-muted-foreground')}>
                  {s.title}
                </span>
              </div>
            )
          })}
        </div>

        <div className="min-h-[280px]">
          {step === 0 && (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1.5">Site Name</label>
                <Input
                  placeholder="My App"
                  value={siteName}
                  onChange={(e) => setSiteName(e.target.value)}
                />
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div className="col-span-2">
                  <label className="block text-sm font-medium mb-1.5">Backend Origin URL</label>
                  <Input
                    placeholder="https://api.internal"
                    value={originUrl}
                    onChange={(e) => setOriginUrl(e.target.value)}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1.5">Port</label>
                  <Input
                    placeholder="8080"
                    value={originPort}
                    onChange={(e) => setOriginPort(e.target.value)}
                  />
                </div>
              </div>
            </div>
          )}

          {step === 1 && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium">Auto-Detect Framework</h3>
                <Button size="sm" onClick={handleDetect} disabled={detecting}>
                  {detecting ? 'Scanning...' : 'Detect'}
                </Button>
              </div>
              {detectedStack ? (
                <div className="space-y-3">
                  {DETECTED_STACKS.map((stack) => (
                    <div
                      key={stack.name}
                      className={cn(
                        'flex items-center justify-between p-3 rounded-lg border cursor-pointer transition-colors',
                        detectedStack === stack.name
                          ? 'border-fortress-500 bg-fortress-500/5'
                          : 'border-border hover:bg-accent',
                      )}
                      onClick={() => setDetectedStack(stack.name)}
                    >
                      <div className="flex items-center gap-3">
                        <Code className="w-5 h-5 text-muted-foreground" />
                        <div>
                          <p className="text-sm font-medium">{stack.name}</p>
                          <p className="text-xs text-muted-foreground">v{stack.version} — {stack.confidence}% confidence</p>
                        </div>
                      </div>
                      {detectedStack === stack.name && <Check className="w-4 h-4 text-fortress-400" />}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-12 text-muted-foreground">
                  <Code className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">Click "Detect" to auto-identify the tech stack</p>
                </div>
              )}
            </div>
          )}

          {step === 2 && (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground mb-3">Select rule templates for your site</p>
              {templates.map((t) => (
                <div
                  key={t.id}
                  className={cn(
                    'flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors',
                    t.selected ? 'border-fortress-500/50 bg-fortress-500/5' : 'border-border hover:bg-accent',
                  )}
                  onClick={() => toggleTemplate(t.id)}
                >
                  <div className={cn('w-4 h-4 rounded border-2 flex items-center justify-center', t.selected ? 'bg-fortress-600 border-fortress-600' : 'border-input')}>
                    {t.selected && <Check className="w-3 h-3 text-white" />}
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium">{t.name}</p>
                    <p className="text-xs text-muted-foreground">{t.description}</p>
                  </div>
                  <Badge variant="outline" className="text-[10px]">{t.id}</Badge>
                </div>
              ))}
            </div>
          )}

          {step === 3 && (
            <div className="space-y-4">
              <h3 className="text-sm font-medium">DNS Configuration</h3>
              <div className="bg-muted p-4 rounded-lg space-y-3">
                <p className="text-sm">Update your DNS records to point your domain to FortressWAF:</p>
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-sm">
                    <Badge variant="secondary" className="font-mono">Type</Badge>
                    <span className="font-mono text-fortress-400">CNAME</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <Badge variant="secondary" className="font-mono">Name</Badge>
                    <span className="font-mono">@</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <Badge variant="secondary" className="font-mono">Value</Badge>
                    <span className="font-mono text-fortress-400">proxy.fortresswaf.io</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <Badge variant="secondary" className="font-mono">TTL</Badge>
                    <span className="font-mono">300 (5 min)</span>
                  </div>
                </div>
              </div>
              <div className="bg-yellow-500/10 border border-yellow-500/20 p-3 rounded-lg">
                <p className="text-xs text-yellow-500">DNS propagation may take up to 48 hours. SSL/TLS certificates will be provisioned automatically.</p>
              </div>
            </div>
          )}

          {step === 4 && (
            <div className="space-y-4">
              <h3 className="text-sm font-medium">Site Summary</h3>
              <div className="bg-muted p-4 rounded-lg space-y-3">
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Site Name</span>
                  <span className="text-sm font-medium">{siteName || 'Unnamed'}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Origin URL</span>
                  <span className="text-sm font-mono">{originUrl || '—'}:{originPort}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Framework</span>
                  <span className="text-sm">{detectedStack || 'Manual config'}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Rule Templates</span>
                  <span className="text-sm">{templates.filter((t) => t.selected).length} selected</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">DNS Proxy</span>
                  <span className="text-sm text-fortress-400">proxy.fortresswaf.io</span>
                </div>
              </div>
              <Button className="w-full bg-fortress-600 hover:bg-fortress-500" size="lg" onClick={handleFinish}>
                <Zap className="w-4 h-4 mr-2" /> Go Live
              </Button>
            </div>
          )}
        </div>

        <div className="flex justify-between pt-4 border-t border-border">
          <Button variant="ghost" onClick={() => step > 0 ? setStep(step - 1) : handleClose()}>
            {step > 0 ? <><ChevronLeft className="w-4 h-4 mr-1" /> Back</> : 'Cancel'}
          </Button>
          <div className="text-xs text-muted-foreground self-center">
            Step {step + 1} of {STEPS.length}
          </div>
          {step < STEPS.length - 1 && (
            <Button onClick={() => setStep(step + 1)} disabled={!canProceed()}>
              Next <ChevronRight className="w-4 h-4 ml-1" />
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
