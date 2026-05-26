'use client'

import * as React from 'react'
import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import {
  LayoutDashboard, Globe, Shield, ScrollText, BarChart3,
  Syringe, Settings, ShieldCheck, Menu, X, Search,
  Bell, Sun, Moon, ChevronDown, LogOut, User, Key,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem,
  DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useTheme } from '@/components/theme-provider'
import { useToast } from '@/components/ui/toast'
import { cn } from '@/lib/utils'
import { WebSocketProvider } from '@/components/ui/websocket-provider'
import { setToken } from '@/lib/api'

interface NavItem {
  label: string
  href: string
  icon: React.ReactNode
  roles?: string[]
}

const navItems: NavItem[] = [
  { label: 'Overview', href: '/dashboard', icon: <LayoutDashboard className="w-4 h-4" /> },
  { label: 'Sites', href: '/dashboard/sites', icon: <Globe className="w-4 h-4" /> },
  { label: 'Rules', href: '/dashboard/rules', icon: <Shield className="w-4 h-4" /> },
  { label: 'Logs', href: '/dashboard/logs', icon: <ScrollText className="w-4 h-4" /> },
  { label: 'Analytics', href: '/dashboard/analytics', icon: <BarChart3 className="w-4 h-4" /> },
  { label: 'Patches', href: '/dashboard/patches', icon: <Syringe className="w-4 h-4" /> },
  { label: 'Settings', href: '/dashboard/settings', icon: <Settings className="w-4 h-4" /> },
  { label: 'Admin', href: '/dashboard/admin', icon: <ShieldCheck className="w-4 h-4" />, roles: ['admin'] },
]

function Avatar({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn('relative flex h-8 w-8 shrink-0 overflow-hidden border-2 border-foreground', className)}>{children}</div>
}

function AvatarFallback({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn('flex h-full w-full items-center justify-center bg-secondary text-secondary-foreground text-xs font-black', className)}>{children}</div>
}

function DashboardShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const router = useRouter()
  const { theme, setTheme } = useTheme()
  const { toast } = useToast()
  const [sidebarOpen, setSidebarOpen] = React.useState(false)

  const userRole = 'admin'
  const filteredNav = navItems.filter((item) => !item.roles || item.roles.includes(userRole))

  function handleLogout() {
    setToken(null)
    toast({ title: 'Logged out', description: 'You have been signed out successfully.' })
    router.push('/')
  }

  return (
    <div className="min-h-screen bg-background">
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 w-64 bg-card border-r-2 border-foreground transform transition-transform duration-200 ease-in-out lg:translate-x-0 lg:static lg:z-auto',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        <div className="flex items-center gap-3 px-6 h-16 border-b-2 border-foreground">
          <div className="flex items-center justify-center w-8 h-8 border-2 border-foreground bg-primary shadow-brutal-sm">
            <Shield className="w-4 h-4 text-primary-foreground" />
          </div>
          <span className="font-black text-foreground uppercase tracking-tight">FortressWAF</span>
        </div>

        <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
          {filteredNav.map((item) => {
            const isActive = pathname === item.href || (item.href !== '/dashboard' && pathname.startsWith(item.href))
            return (
              <Link
                key={item.href}
                href={item.href}
                onClick={() => setSidebarOpen(false)}
                className={cn(
                  'flex items-center gap-3 px-3 py-2.5 text-sm font-bold uppercase tracking-wide border-2 border-transparent transition-all',
                  isActive
                    ? 'bg-primary text-primary-foreground border-foreground shadow-brutal-sm'
                    : 'text-muted-foreground hover:bg-muted hover:border-foreground',
                )}
              >
                {item.icon}
                {item.label}
              </Link>
            )
          })}
        </nav>

        <div className="px-3 py-4 border-t-2 border-foreground">
          <div className="flex items-center gap-3 px-3 py-2">
            <Avatar>
              <AvatarFallback>AD</AvatarFallback>
            </Avatar>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-bold text-foreground truncate">Admin User</p>
              <p className="text-xs text-muted-foreground truncate">admin@fortresswaf.io</p>
            </div>
          </div>
        </div>
      </aside>

      {sidebarOpen && (
        <div className="fixed inset-0 z-40 bg-foreground/50 lg:hidden" onClick={() => setSidebarOpen(false)} />
      )}

      <div className="lg:pl-64">
        <header className="sticky top-0 z-30 flex items-center gap-4 px-4 h-16 bg-card border-b-2 border-foreground">
          <Button variant="ghost" size="icon" className="lg:hidden" onClick={() => setSidebarOpen(true)}>
            {sidebarOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </Button>

          <div className="hidden sm:flex relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Search sites, rules, logs..."
              className="pl-9 bg-background"
            />
          </div>

          <div className="flex items-center gap-2 ml-auto">
            <Button
              variant="ghost"
              size="icon"
              className="relative"
              onClick={() => toast({ title: 'Notifications', description: 'No new notifications' })}
            >
              <Bell className="w-5 h-5" />
              <span className="absolute top-1 right-1 w-2 h-2 bg-destructive border-2 border-foreground" />
            </Button>

            <Button variant="ghost" size="icon" onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}>
              {theme === 'dark' ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
            </Button>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="flex items-center gap-2 px-2">
                  <Avatar>
                    <AvatarFallback>AD</AvatarFallback>
                  </Avatar>
                  <ChevronDown className="w-4 h-4 text-muted-foreground hidden sm:block" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                <DropdownMenuLabel>My Account</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => router.push('/dashboard/settings')}>
                  <User className="w-4 h-4 mr-2" /> Profile
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => router.push('/dashboard/settings')}>
                  <Key className="w-4 h-4 mr-2" /> API Keys
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={handleLogout} className="text-destructive">
                  <LogOut className="w-4 h-4 mr-2" /> Sign out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </header>

        <main className="p-6">
          {children}
        </main>
      </div>
    </div>
  )
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <WebSocketProvider>
      <DashboardShell>{children}</DashboardShell>
    </WebSocketProvider>
  )
}
