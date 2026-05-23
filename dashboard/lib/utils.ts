import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date
  return d.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

export function formatNumber(num: number): string {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`
  return num.toString()
}

export function getSeverityColor(severity: string): string {
  const map: Record<string, string> = {
    critical: 'text-red-500 bg-red-500/10 border-red-500/20',
    high: 'text-orange-500 bg-orange-500/10 border-orange-500/20',
    medium: 'text-yellow-500 bg-yellow-500/10 border-yellow-500/20',
    low: 'text-blue-500 bg-blue-500/10 border-blue-500/20',
    info: 'text-green-500 bg-green-500/10 border-green-500/20',
  }
  return map[severity.toLowerCase()] || map.info
}

export function getAttackColor(action: string): string {
  const map: Record<string, string> = {
    blocked: 'text-red-500',
    allowed: 'text-green-500',
    challenged: 'text-yellow-500',
    logged: 'text-blue-500',
  }
  return map[action.toLowerCase()] || 'text-muted-foreground'
}

export function getStatusColor(status: string): string {
  const map: Record<string, string> = {
    online: 'text-green-500',
    offline: 'text-red-500',
    degraded: 'text-yellow-500',
    enabled: 'text-green-500',
    disabled: 'text-muted-foreground',
    active: 'text-green-500',
    inactive: 'text-muted-foreground',
  }
  return map[status.toLowerCase()] || 'text-muted-foreground'
}

export function getStatusDot(status: string): string {
  const map: Record<string, string> = {
    online: 'bg-green-500',
    offline: 'bg-red-500',
    degraded: 'bg-yellow-500',
    enabled: 'bg-green-500',
    disabled: 'bg-gray-400',
    active: 'bg-green-500',
    testing: 'bg-yellow-500',
    draft: 'bg-gray-400',
    deployed: 'bg-green-500',
    expired: 'bg-red-500',
  }
  return map[status.toLowerCase()] || 'bg-gray-400'
}

export function classNames(...classes: (string | boolean | undefined | null)[]): string {
  return classes.filter(Boolean).join(' ')
}
