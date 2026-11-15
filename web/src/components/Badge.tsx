import type { ReactNode } from 'react'
import { cn } from '../lib/cn'
import type { Tone } from '../lib/status'

// Soft: a tint of the tone behind text in the tone. No outline, so a row of
// badges reads as labels rather than as buttons.
const tones: Record<Tone, string> = {
  neutral: 'bg-surface-3 text-muted',
  accent: 'bg-accent/14 text-accent',
  ok: 'bg-ok/14 text-ok',
  warn: 'bg-warn/14 text-warn',
  bad: 'bg-bad/14 text-bad',
  info: 'bg-info/14 text-info',
}

/** Opaque fills, for badges that sit on top of cover art. */
const solidTones: Record<Tone, string> = {
  neutral: 'bg-surface-3 text-fg',
  accent: 'bg-accent text-accent-fg',
  ok: 'bg-ok text-on-status',
  warn: 'bg-warn text-on-status',
  bad: 'bg-bad text-on-status',
  info: 'bg-info text-on-status',
}

const dots: Record<Tone, string> = {
  neutral: 'bg-faint',
  accent: 'bg-accent',
  ok: 'bg-ok',
  warn: 'bg-warn',
  bad: 'bg-bad',
  info: 'bg-info',
}

export function Badge({
  tone = 'neutral',
  solid,
  dot,
  children,
  className,
  title,
}: {
  tone?: Tone
  /** Use over images, where the tinted style is unreadable. */
  solid?: boolean
  /** A status: a coloured dot and plain text, with no chip behind it. For dense tables. */
  dot?: boolean
  children: ReactNode
  className?: string
  title?: string
}) {
  if (dot) {
    return (
      <span title={title} className={cn('inline-flex items-center gap-1.5 whitespace-nowrap text-muted', className)}>
        <span className={cn('size-1.5 shrink-0 rounded-full', dots[tone])} aria-hidden />
        {children}
      </span>
    )
  }
  return (
    <span
      title={title}
      className={cn(
        'inline-flex h-5 items-center gap-1 whitespace-nowrap rounded-chip px-1.5 text-xs font-medium',
        solid ? cn(solidTones[tone], 'shadow-overlay') : tones[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}
