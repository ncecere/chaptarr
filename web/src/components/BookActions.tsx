import { Search } from 'lucide-react'
import type { Book } from '../lib/api'
import { useSearchBook, useUpdateBook } from '../lib/mutations'
import { Button, IconButton } from './Button'
import { Toggle } from './Toggle'

/** Monitored switch for one book; shows the requested value while the request is in flight. */
export function MonitorToggle({ book, size = 'sm' }: { book: Pick<Book, 'id' | 'title' | 'monitored'>; size?: 'sm' | 'md' }) {
  const m = useUpdateBook()
  const value = m.isPending ? m.variables.monitored : book.monitored
  return (
    <Toggle
      size={size}
      checked={value}
      label={`${value ? 'Unmonitor' : 'Monitor'} ${book.title}`}
      onChange={(monitored) => m.mutate({ id: book.id, monitored })}
    />
  )
}

/** Automatic search for one book. Icon-only by default, labelled when `label` is set. */
export function SearchBookButton({ bookId, title, label }: { bookId: number; title?: string; label?: string }) {
  const m = useSearchBook()
  if (label) {
    return (
      <Button
        icon={<Search className="size-4" />}
        loading={m.isPending}
        onClick={() => m.mutate(bookId)}
      >
        {label}
      </Button>
    )
  }
  return (
    <IconButton
      size="sm"
      label={title ? `Search for ${title}` : 'Automatic search'}
      loading={m.isPending}
      onClick={() => m.mutate(bookId)}
    >
      <Search className="size-4" />
    </IconButton>
  )
}
