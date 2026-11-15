import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { cn } from '../lib/cn'

const tones = {
  primary: 'bg-accent text-accent-fg shadow-raised hover:bg-accent-hover',
  ok: 'border border-ok/25 bg-ok/10 text-ok hover:bg-ok/15',
}

/** A router link that looks like a button, for actions that are navigation. */
export function LinkButton({
  to,
  tone = 'primary',
  icon,
  children,
  className,
}: {
  to: string
  tone?: keyof typeof tones
  icon?: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <Link
      to={to}
      className={cn(
        'inline-flex h-8 shrink-0 items-center justify-center gap-1.5 whitespace-nowrap rounded-control px-3 text-sm font-medium transition-colors',
        tones[tone],
        className,
      )}
    >
      {icon}
      {children}
    </Link>
  )
}
