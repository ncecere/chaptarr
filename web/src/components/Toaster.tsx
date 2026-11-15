import { CircleAlert, CircleCheck, Info, X } from 'lucide-react'
import { cn } from '../lib/cn'
import { dismissToast, useToasts, type ToastKind } from '../lib/toast'

const kinds: Record<ToastKind, { icon: typeof Info; cls: string }> = {
  success: { icon: CircleCheck, cls: 'text-ok' },
  error: { icon: CircleAlert, cls: 'text-bad' },
  info: { icon: Info, cls: 'text-info' },
}

export function Toaster() {
  const toasts = useToasts()
  return (
    <div
      aria-live="polite"
      className="pointer-events-none fixed inset-x-3 bottom-3 z-[60] flex flex-col items-end gap-2 sm:inset-x-auto sm:right-4 sm:bottom-4"
    >
      {toasts.map((t) => {
        const { icon: Icon, cls } = kinds[t.kind]
        return (
          <div
            key={t.id}
            role={t.kind === 'error' ? 'alert' : 'status'}
            className="pointer-events-auto flex w-full max-w-sm items-start gap-2.5 rounded-card border border-line bg-surface py-2.5 pr-2 pl-3 text-sm shadow-overlay animate-toast-in"
          >
            <Icon className={cn('mt-0.5 size-4 shrink-0', cls)} />
            <div className="min-w-0 flex-1 break-words">{t.message}</div>
            <button
              type="button"
              aria-label="Dismiss"
              onClick={() => dismissToast(t.id)}
              className="rounded-chip p-1 text-muted transition-colors hover:bg-surface-2 hover:text-fg"
            >
              <X className="size-3.5" />
            </button>
          </div>
        )
      })}
    </div>
  )
}
