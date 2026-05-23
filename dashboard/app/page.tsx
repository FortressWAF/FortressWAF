'use client'

import * as React from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Shield, Eye, EyeOff, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useToast } from '@/components/ui/toast'
import { api, setToken } from '@/lib/api'
import { cn } from '@/lib/utils'
import { useRouter } from 'next/navigation'

const loginSchema = z.object({
  email: z.string().email('Please enter a valid email'),
  password: z.string().min(6, 'Password must be at least 6 characters'),
})

type LoginForm = z.infer<typeof loginSchema>

export default function LoginPage() {
  const router = useRouter()
  const { toast } = useToast()
  const [showPassword, setShowPassword] = React.useState(false)
  const [isLoading, setIsLoading] = React.useState(false)
  const [ssoLoading, setSsoLoading] = React.useState(false)

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
  })

  async function onSubmit(data: LoginForm) {
    setIsLoading(true)
    try {
      const response = await api.auth.login(data.email, data.password)
      setToken(response.token)
      toast({ title: 'Welcome back!', description: 'Redirecting to dashboard...', variant: 'success' })
      router.push('/dashboard')
    } catch (err: unknown) {
      const error = err as { message?: string }
      toast({ title: 'Login failed', description: error.message || 'Invalid credentials', variant: 'destructive' })
    } finally {
      setIsLoading(false)
    }
  }

  async function handleSSO() {
    setSsoLoading(true)
    try {
      const response = await api.auth.sso('saml')
      setToken(response.token)
      router.push('/dashboard')
    } catch {
      toast({ title: 'SSO login failed', description: 'Could not authenticate with SSO', variant: 'destructive' })
    } finally {
      setSsoLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-900 via-fortress-950 to-slate-900">
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-fortress-500/10 via-transparent to-transparent" />
      <div className="relative w-full max-w-md px-8">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-fortress-600 shadow-lg shadow-fortress-500/25 mb-4">
            <Shield className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-white">FortressWAF</h1>
          <p className="text-slate-400 mt-1">Enterprise Web Application Firewall</p>
        </div>

        <div className="bg-white/5 backdrop-blur-xl border border-white/10 rounded-2xl p-8 shadow-2xl">
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-1.5">
                Email address
              </label>
              <Input
                id="email"
                type="email"
                placeholder="admin@company.com"
                className="bg-white/5 border-white/10 text-white placeholder:text-slate-500 focus:border-fortress-500"
                {...register('email')}
              />
              {errors.email && <p className="text-red-400 text-xs mt-1">{errors.email.message}</p>}
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium text-slate-300 mb-1.5">
                Password
              </label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="••••••••"
                  className="bg-white/5 border-white/10 text-white placeholder:text-slate-500 focus:border-fortress-500 pr-10"
                  {...register('password')}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-300"
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              {errors.password && <p className="text-red-400 text-xs mt-1">{errors.password.message}</p>}
            </div>

            <div className="flex items-center justify-end">
              <button type="button" className="text-sm text-fortress-400 hover:text-fortress-300 transition-colors">
                Forgot password?
              </button>
            </div>

            <Button
              type="submit"
              disabled={isLoading}
              className="w-full bg-fortress-600 hover:bg-fortress-500 text-white shadow-lg shadow-fortress-500/25"
            >
              {isLoading ? <Loader2 className="w-4 h-4 animate-spin mr-2" /> : null}
              Sign in
            </Button>
          </form>

          <div className="relative my-6">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-white/10" />
            </div>
            <div className="relative flex justify-center text-xs">
              <span className="px-2 bg-transparent text-slate-500">or continue with</span>
            </div>
          </div>

          <Button
            type="button"
            variant="outline"
            disabled={ssoLoading}
            onClick={handleSSO}
            className="w-full border-white/10 text-slate-300 hover:bg-white/5 hover:text-white"
          >
            {ssoLoading ? <Loader2 className="w-4 h-4 animate-spin mr-2" /> : null}
            Sign in with SSO
          </Button>
        </div>

        <p className="text-center text-xs text-slate-600 mt-6">
          &copy; {new Date().getFullYear()} FortressWAF. All rights reserved.
        </p>
      </div>
    </div>
  )
}
