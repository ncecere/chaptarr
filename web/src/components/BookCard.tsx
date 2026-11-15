import { EyeOff } from 'lucide-react'
import { Link } from 'react-router-dom'
import type { Book } from '../lib/api'
import { Cover } from './Cover'
import { BookStatusBadge } from './StatusBadge'

export function BookCard({ book }: { book: Book }) {
  return (
    <Link to={`/books/${book.id}`} className="group block min-w-0">
      <div className="relative">
        <Cover
          src={book.coverUrl}
          alt={`Cover of ${book.title}`}
          className="transition-transform duration-200 group-hover:-translate-y-0.5 group-hover:ring-accent/50"
        />
        <BookStatusBadge status={book.status} className="absolute bottom-1.5 left-1.5 backdrop-blur-sm" />
        {!book.monitored && (
          <span
            title="Not monitored"
            className="absolute top-1.5 right-1.5 grid size-6 place-items-center rounded-full bg-black/60 text-muted backdrop-blur-sm"
          >
            <EyeOff className="size-3.5" />
          </span>
        )}
      </div>
      <div className="mt-2 min-w-0">
        <div className="truncate text-sm font-medium transition-colors group-hover:text-accent" title={book.title}>
          {book.title}
        </div>
        <div className="truncate text-xs text-muted">{book.authorName}</div>
      </div>
    </Link>
  )
}
