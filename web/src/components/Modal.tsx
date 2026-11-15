import { X } from 'lucide-react'
import { useEffect, useId, useRef, type KeyboardEvent, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '../lib/cn'
import { IconButton } from './Button'

const sizes = {
  sm: 'max-w-md',
  md: 'max-w-xl',
  lg: 'max-w-3xl',
  xl: 'max-w-6xl',
}

interface ModalProps {
  open: boolean
  onClose: () => void
  title: ReactNode
  children: ReactNode
  footer?: ReactNode
  size?: keyof typeof sizes
  /** While busy, Escape and backdrop clicks do not close the dialog. */
  busy?: boolean
}

const FOCUSABLE =
  'a[href],button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])'

export function Modal({ open, onClose, title, children, footer, size = 'md', busy }: ModalProps) {
  const panelRef = useRef<HTMLDivElement>(null)
  const titleId = useId()
  const onCloseRef = useRef(onClose)
  const busyRef = useRef(busy)
  useEffect(() => {
    onCloseRef.current = onClose
    busyRef.current = busy
  })

  useEffect(() => {
    if (!open) return
    const previous = document.activeElement as HTMLElement | null
    const overflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const panel = panelRef.current
    if (panel && !panel.contains(document.activeElement)) panel.focus()
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'Escape' && !busyRef.current) onCloseRef.current()
    }
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = overflow
      previous?.focus?.()
    }
  }, [open])

  if (!open) return null

  // Keep Tab inside the dialog.
  const trapTab = (e: KeyboardEvent) => {
    if (e.key !== 'Tab' || !panelRef.current) return
    const items = Array.from(panelRef.current.querySelectorAll<HTMLElement>(FOCUSABLE))
    if (items.length === 0) return
    const first = items[0]
    const last = items[items.length - 1]
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
  }

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-scrim p-0 backdrop-blur-[3px] animate-fade-in sm:items-center sm:p-4"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && !busy) onClose()
      }}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        onKeyDown={trapTab}
        className={cn(
          'flex max-h-[92vh] w-full flex-col rounded-t-dialog border border-line bg-surface shadow-overlay outline-none animate-pop-in sm:rounded-dialog',
          sizes[size],
        )}
      >
        <div className="flex items-center justify-between gap-4 border-b border-line py-3 pr-3 pl-5">
          <h2 id={titleId} className="min-w-0 truncate text-base font-semibold">
            {title}
          </h2>
          <IconButton label="Close" size="sm" onClick={onClose} disabled={busy}>
            <X className="size-4" />
          </IconButton>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">{children}</div>
        {footer && (
          <div className="flex flex-wrap items-center justify-end gap-2 rounded-b-dialog border-t border-line bg-surface-2/50 px-5 py-3">
            {footer}
          </div>
        )}
      </div>
    </div>,
    document.body,
  )
}
