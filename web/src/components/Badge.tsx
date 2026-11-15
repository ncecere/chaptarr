import type { ReactNode } from 'react'
import { cn } from '../lib/cn'
import type { Tone } from '../lib/status'

const tones: Record<Tone, string> = {
  neutral: 'bg-surface-3 text-muted ring-line-strong',
  accent: 'bg-accent/15 text-accent ring-accent/30',
  ok: 'bg-ok/12 text-ok ring-ok/25',
  warn: 'bg-warn/12 text-warn ring-warn/25',
  bad: 'bg-bad/12 text-bad ring-bad/25',
  info: 'bg-info/12 text-info ring-info/25',
}

/** Opaque fills, for badges that sit on top of cover art. */
const solidTones: Record<Tone, string> = {
  neutral: 'bg-surface-3 text-fg ring-black/40',
  accent: 'bg-accent text-canvas ring-black/40',
  ok: 'bg-ok text-canvas ring-black/40',
  warn: 'bg-warn text-canvas ring-black/40',
  bad: 'bg-bad text-canvas ring-black/40',
  info: 'bg-info text-canvas ring-black/40',
}

export function Badge({
  tone = 'neutral',
  solid,
  children,
  className,
  title,
}: {
  tone?: Tone
  /** Use over images, where the tinted style is unreadable. */
  solid?: boolean
  children: ReactNode
  className?: string
  title?: string
}) {
  return (
    <span
      title={title}
      className={cn(
        'inline-flex items-center gap-1 whitespace-nowrap rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset',
        solid ? cn(solidTones[tone], 'font-semibold shadow-md shadow-black/50') : tones[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}
