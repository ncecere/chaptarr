import { cn } from '../lib/cn'

const tones = { accent: 'bg-accent', ok: 'bg-ok', warn: 'bg-warn', bad: 'bg-bad' }

export function ProgressBar({
  value,
  tone = 'accent',
  className,
}: {
  /** 0..1 */
  value: number
  tone?: keyof typeof tones
  className?: string
}) {
  const pct = Math.min(1, Math.max(0, Number.isFinite(value) ? value : 0)) * 100
  return (
    <div
      role="progressbar"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(pct)}
      className={cn('h-1 w-full overflow-hidden rounded-full bg-surface-3', className)}
    >
      <div
        className={cn('h-full rounded-full transition-[width] duration-500', tones[tone])}
        style={{ width: `${pct}%` }}
      />
    </div>
  )
}
