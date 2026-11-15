import type { ReactNode } from 'react'
import { cn } from '../lib/cn'

/** Responsive grid for cover/photo cards. */
export function CoverGrid({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn('grid grid-cols-[repeat(auto-fill,minmax(9.5rem,1fr))] gap-x-4 gap-y-6', className)}>
      {children}
    </div>
  )
}

export function CoverGridSkeleton({ count = 12 }: { count?: number }) {
  return (
    <CoverGrid>
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="animate-pulse">
          <div className="aspect-square rounded-control bg-surface-2" />
          <div className="mt-2 h-3.5 w-4/5 rounded-chip bg-surface-2" />
          <div className="mt-1.5 h-3 w-1/2 rounded-chip bg-surface-2" />
        </div>
      ))}
    </CoverGrid>
  )
}
