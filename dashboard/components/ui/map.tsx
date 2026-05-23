'use client'

import * as React from 'react'
import { cn } from '@/lib/utils'
import type { GeoData } from '@/types'

interface WorldMapHeatmapProps {
  data: GeoData[]
  className?: string
  title?: string
}

const COUNTRY_COORDS: Record<string, { x: number; y: number }> = {
  US: { x: 15, y: 35 },
  CN: { x: 72, y: 32 },
  RU: { x: 68, y: 25 },
  BR: { x: 40, y: 65 },
  IN: { x: 76, y: 45 },
  GB: { x: 48, y: 20 },
  DE: { x: 52, y: 22 },
  FR: { x: 50, y: 26 },
  JP: { x: 85, y: 30 },
  KR: { x: 82, y: 32 },
  SG: { x: 78, y: 55 },
  AU: { x: 88, y: 72 },
  CA: { x: 12, y: 25 },
  MX: { x: 18, y: 45 },
  AR: { x: 38, y: 70 },
  ZA: { x: 55, y: 68 },
  NG: { x: 55, y: 50 },
  EG: { x: 58, y: 38 },
  SA: { x: 62, y: 42 },
  IR: { x: 65, y: 33 },
  PK: { x: 70, y: 40 },
  BD: { x: 76, y: 45 },
  ID: { x: 80, y: 58 },
  PH: { x: 84, y: 50 },
  VN: { x: 78, y: 48 },
  TH: { x: 76, y: 52 },
  TW: { x: 83, y: 42 },
  HK: { x: 81, y: 45 },
  SE: { x: 53, y: 15 },
  NO: { x: 52, y: 12 },
  NL: { x: 50, y: 22 },
  IT: { x: 52, y: 30 },
  ES: { x: 48, y: 32 },
  PL: { x: 55, y: 22 },
  UA: { x: 58, y: 24 },
  TR: { x: 60, y: 32 },
  IL: { x: 62, y: 36 },
  AE: { x: 64, y: 42 },
}

function WorldMapHeatmap({ data, className, title }: WorldMapHeatmapProps) {
  const maxAttacks = Math.max(...data.map((d) => d.attacks), 1)

  function getIntensity(attacks: number): string {
    const ratio = attacks / maxAttacks
    if (ratio > 0.8) return 'bg-red-500/90'
    if (ratio > 0.6) return 'bg-orange-500/80'
    if (ratio > 0.4) return 'bg-yellow-500/70'
    if (ratio > 0.2) return 'bg-lime-500/50'
    return 'bg-green-500/30'
  }

  function getRadius(attacks: number): number {
    const ratio = attacks / maxAttacks
    return Math.max(4, ratio * 16)
  }

  return (
    <div className={cn('rounded-lg border bg-card p-6 shadow-sm', className)}>
      {title && <h3 className="text-sm font-medium text-foreground mb-4">{title}</h3>}
      <svg viewBox="0 0 100 75" className="w-full h-auto" xmlns="http://www.w3.org/2000/svg">
        <rect width="100" height="75" fill="hsl(var(--muted))" rx="4" />
        <path
          d="M5 35 L15 20 L25 18 L35 15 L45 20 L55 18 L65 22 L75 20 L85 25 L90 30 L95 35 L95 55 L85 60 L75 65 L65 68 L55 65 L45 68 L35 65 L25 62 L15 60 L5 55 Z"
          fill="hsl(var(--background))"
          opacity="0.6"
        />
        {data.map((point) => {
          const coord = COUNTRY_COORDS[point.code]
          if (!coord) return null
          const r = getRadius(point.attacks)
          return (
            <g key={point.code}>
              <circle
                cx={coord.x}
                cy={coord.y}
                r={r}
                className={getIntensity(point.attacks)}
                opacity={0.8}
              />
              <title>{`${point.country}: ${point.attacks.toLocaleString()} attacks`}</title>
            </g>
          )
        })}
      </svg>
      <div className="flex items-center justify-between mt-4 text-xs text-muted-foreground">
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-green-500/50" />
          <span>Low</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-yellow-500/70" />
          <span>Medium</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-orange-500/80" />
          <span>High</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-red-500/90" />
          <span>Critical</span>
        </div>
      </div>
    </div>
  )
}

export { WorldMapHeatmap }
