import { BookOpen, Layers, User } from 'lucide-react'
import { useState } from 'react'
import { cn } from '../lib/cn'

interface CoverProps {
  src?: string | null
  alt: string
  kind?: 'book' | 'author' | 'series'
  className?: string
}

/** Square cover/photo from an external URL, with a placeholder when missing or broken. */
export function Cover({ src, alt, kind = 'book', className }: CoverProps) {
  const [failedSrc, setFailedSrc] = useState<string | null>(null)
  const showImage = !!src && failedSrc !== src
  const Icon = kind === 'author' ? User : kind === 'series' ? Layers : BookOpen
  return (
    <div
      className={cn(
        'relative aspect-square shrink-0 overflow-hidden bg-surface-2 ring-1 ring-inset ring-line',
        kind === 'author' ? 'rounded-full' : 'rounded-lg',
        className,
      )}
    >
      {showImage ? (
        <img
          src={src}
          alt={alt}
          loading="lazy"
          decoding="async"
          referrerPolicy="no-referrer"
          onError={() => setFailedSrc(src)}
          className="size-full object-cover"
        />
      ) : (
        <div className="absolute inset-0 grid place-items-center text-faint" role="img" aria-label={alt}>
          <Icon className="size-1/3" strokeWidth={1.5} />
        </div>
      )}
    </div>
  )
}
