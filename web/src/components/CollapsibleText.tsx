import { useState } from 'react'
import { cn } from '../lib/cn'

/** Plain text that clamps to a few lines with a Show more / Show less toggle. */
export function CollapsibleText({
  text,
  className,
  threshold = 320,
}: {
  text: string
  className?: string
  /** Texts shorter than this never collapse. */
  threshold?: number
}) {
  const [open, setOpen] = useState(false)
  const long = text.length > threshold
  return (
    <div className={className}>
      <p className={cn('whitespace-pre-line text-sm leading-relaxed text-muted', long && !open && 'line-clamp-4')}>
        {text}
      </p>
      {long && (
        <button
          type="button"
          onClick={() => setOpen(!open)}
          className="mt-1 text-xs font-medium text-accent hover:underline"
          aria-expanded={open}
        >
          {open ? 'Show less' : 'Show more'}
        </button>
      )}
    </div>
  )
}
