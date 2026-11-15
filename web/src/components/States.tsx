import { CircleAlert } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '../lib/cn'
import { errorMessage } from '../lib/queryClient'
import { Button } from './Button'
import { Spinner } from './Spinner'

export function LoadingState({ label = 'Loading…', className }: { label?: string; className?: string }) {
  return (
    <div
      role="status"
      className={cn('flex items-center justify-center gap-2.5 py-16 text-sm text-muted', className)}
    >
      <Spinner className="size-5 text-accent" />
      {label}
    </div>
  )
}

export function EmptyState({
  icon,
  title,
  description,
  action,
  className,
}: {
  icon?: ReactNode
  title: string
  description?: ReactNode
  action?: ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        'flex flex-col items-center rounded-xl border border-dashed border-line-strong px-6 py-14 text-center',
        className,
      )}
    >
      {icon && <div className="mb-3 text-faint [&>svg]:size-10 [&>svg]:stroke-1">{icon}</div>}
      <h3 className="text-base font-medium">{title}</h3>
      {description && <p className="mt-1 max-w-md text-sm text-muted">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  )
}

export function ErrorState({
  error,
  title = 'Something went wrong',
  onRetry,
  className,
}: {
  error?: unknown
  title?: string
  onRetry?: () => void
  className?: string
}) {
  return (
    <div
      role="alert"
      className={cn(
        'flex flex-col items-center rounded-xl border border-bad/30 bg-bad/5 px-6 py-12 text-center',
        className,
      )}
    >
      <CircleAlert className="mb-3 size-9 text-bad" strokeWidth={1.5} />
      <h3 className="text-base font-medium">{title}</h3>
      {error !== undefined && <p className="mt-1 max-w-lg text-sm text-muted">{errorMessage(error)}</p>}
      {onRetry && (
        <Button className="mt-4" onClick={onRetry}>
          Try again
        </Button>
      )}
    </div>
  )
}
