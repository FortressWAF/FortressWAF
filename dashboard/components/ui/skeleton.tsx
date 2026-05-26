import * as React from 'react'
import { cn } from '@/lib/utils'

function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('animate-pulse bg-muted border-2 border-foreground', className)} {...props} />
}

function CardSkeleton() {
  return (
    <div className="border-2 border-foreground bg-card p-5 shadow-brutal">
      <Skeleton className="h-4 w-[120px] mb-2" />
      <Skeleton className="h-8 w-[80px] mb-1" />
      <Skeleton className="h-3 w-[100px]" />
    </div>
  )
}

function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="space-y-0 border-2 border-foreground bg-card">
      <div className="flex gap-4 p-4 bg-muted border-b-2 border-foreground">
        <Skeleton className="h-4 flex-1" />
        <Skeleton className="h-4 flex-1" />
        <Skeleton className="h-4 flex-1" />
        <Skeleton className="h-4 flex-1" />
      </div>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex gap-4 p-4 border-b-2 border-foreground last:border-b-0">
          <Skeleton className="h-4 flex-1" />
          <Skeleton className="h-4 flex-1" />
          <Skeleton className="h-4 flex-1" />
          <Skeleton className="h-4 flex-1" />
        </div>
      ))}
    </div>
  )
}

export { Skeleton, CardSkeleton, TableSkeleton }
