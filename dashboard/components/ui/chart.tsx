'use client'

import * as React from 'react'
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  LineChart,
  Line,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  type TooltipProps,
} from 'recharts'
import { cn } from '@/lib/utils'

const COLORS = ['#06b6d4', '#f59e0b', '#ef4444', '#10b981', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316']

interface ChartContainerProps {
  title?: string
  subtitle?: string
  className?: string
  children: React.ReactNode
}

function ChartContainer({ title, subtitle, className, children }: ChartContainerProps) {
  return (
    <div className={cn('border-2 border-foreground bg-card p-5 shadow-brutal', className)}>
      {title && <h3 className="text-sm font-black uppercase tracking-tight text-foreground">{title}</h3>}
      {subtitle && <p className="text-xs text-muted-foreground font-medium mt-1 mb-4">{subtitle}</p>}
      {children}
    </div>
  )
}

interface SparklineProps {
  data: { value: number }[]
  color?: string
  height?: number
}

function Sparkline({ data, color = '#06b6d4', height = 40 }: SparklineProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data} margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id={`gradient-${color}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={color} stopOpacity={0.3} />
            <stop offset="95%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <Area
          type="monotone"
          dataKey="value"
          stroke={color}
          fill={`url(#gradient-${color})`}
          strokeWidth={2}
          dot={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}

interface AreaChartProps {
  data: Record<string, unknown>[]
  xKey: string
  series: { key: string; color?: string; name?: string }[]
  height?: number
  showGrid?: boolean
  showLegend?: boolean
}

function AreaChartComponent({ data, xKey, series, height = 300, showGrid = true, showLegend = true }: AreaChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
        {showGrid && <CartesianGrid strokeDasharray="4 4" stroke="hsl(var(--muted-foreground) / 0.3)" strokeWidth={1.5} />}
        <XAxis dataKey={xKey} tick={{ fontSize: 11, fontWeight: 'bold' }} stroke="hsl(var(--foreground))" tickLine={false} />
        <YAxis tick={{ fontSize: 11, fontWeight: 'bold' }} stroke="hsl(var(--foreground))" tickLine={false} />
        <Tooltip
          contentStyle={{
            backgroundColor: 'hsl(var(--card))',
            border: '2px solid hsl(var(--foreground))',
            borderRadius: 0,
            color: 'hsl(var(--card-foreground))',
            boxShadow: '4px 4px 0px 0px hsl(var(--foreground))',
          }}
        />
        {showLegend && <Legend />}
        {series.map((s) => (
          <Area
            key={s.key}
            type="monotone"
            dataKey={s.key}
            name={s.name || s.key}
            stroke={s.color || '#06b6d4'}
            fill={s.color || '#06b6d4'}
            fillOpacity={0.1}
            strokeWidth={2}
            stackId="1"
          />
        ))}
      </AreaChart>
    </ResponsiveContainer>
  )
}

function BarChartComponent({ data, xKey, series, height = 300, showGrid = true }: AreaChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
        {showGrid && <CartesianGrid strokeDasharray="4 4" stroke="hsl(var(--muted-foreground) / 0.3)" strokeWidth={1.5} />}
        <XAxis dataKey={xKey} tick={{ fontSize: 11, fontWeight: 'bold' }} stroke="hsl(var(--foreground))" tickLine={false} />
        <YAxis tick={{ fontSize: 11, fontWeight: 'bold' }} stroke="hsl(var(--foreground))" tickLine={false} />
        <Tooltip
          contentStyle={{
            backgroundColor: 'hsl(var(--card))',
            border: '2px solid hsl(var(--foreground))',
            borderRadius: 0,
            color: 'hsl(var(--card-foreground))',
            boxShadow: '4px 4px 0px 0px hsl(var(--foreground))',
          }}
        />
        <Legend />
        {series.map((s, i) => (
          <Bar key={s.key} dataKey={s.key} name={s.name || s.key} fill={s.color || COLORS[i % COLORS.length]} />
        ))}
      </BarChart>
    </ResponsiveContainer>
  )
}

interface PieChartProps {
  data: { name: string; value: number; color?: string }[]
  height?: number
  innerRadius?: number
  outerRadius?: number
}

function PieChartComponent({ data, height = 300, innerRadius = 60, outerRadius = 100 }: PieChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <PieChart>
        <Pie
          data={data}
          cx="50%"
          cy="50%"
          innerRadius={innerRadius}
          outerRadius={outerRadius}
          paddingAngle={2}
          dataKey="value"
          stroke="hsl(var(--foreground))"
          strokeWidth={2}
          label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
        >
          {data.map((entry, index) => (
            <Cell key={`cell-${index}`} fill={entry.color || COLORS[index % COLORS.length]} />
          ))}
        </Pie>
        <Tooltip
          contentStyle={{
            backgroundColor: 'hsl(var(--card))',
            border: '2px solid hsl(var(--foreground))',
            borderRadius: 0,
            color: 'hsl(var(--card-foreground))',
            boxShadow: '4px 4px 0px 0px hsl(var(--foreground))',
          }}
        />
      </PieChart>
    </ResponsiveContainer>
  )
}

export { ChartContainer, Sparkline, AreaChartComponent as AreaChart, BarChartComponent as BarChart, PieChartComponent as PieChart, COLORS }
