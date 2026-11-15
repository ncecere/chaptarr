import { Check, Plus } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Badge } from '../../components/Badge'
import { Button } from '../../components/Button'
import { Checkbox } from '../../components/Checkbox'
import { Cover } from '../../components/Cover'
import { LinkButton } from '../../components/LinkButton'
import type { Book, BookResult } from '../../lib/api'
import { formatRuntime, releaseYear } from '../../lib/format'
import { useAddBook } from '../../lib/mutations'
import { metadataSourceMeta } from '../../lib/status'
import { htmlToText } from '../../lib/text'

export function BookResultRow({ book }: { book: BookResult }) {
  const add = useAddBook()
  const [optionsOpen, setOptionsOpen] = useState(false)
  const [monitored, setMonitored] = useState(true)
  const [searchOnAdd, setSearchOnAdd] = useState(true)
  const [added, setAdded] = useState<Book | null>(null)
  const description = useMemo(() => htmlToText(book.description), [book.description])

  const libraryId = added?.id ?? book.libraryBookId
  const year = releaseYear(book.releaseDate)
  const facts = [
    book.seriesName && `${book.seriesName}${book.seriesPosition ? ` #${book.seriesPosition}` : ''}`,
    book.runtimeMinutes > 0 && formatRuntime(book.runtimeMinutes),
    year,
  ].filter(Boolean)

  return (
    <li className="rounded-card border border-line bg-surface p-3 sm:p-4">
      <div className="flex gap-3 sm:gap-4">
        <Cover src={book.coverUrl} alt={`Cover of ${book.title}`} className="size-20 sm:size-24" />
        <div className="min-w-0 flex-1">
          <h3 className="flex flex-wrap items-center gap-x-2 gap-y-1 font-medium leading-snug">
            {book.title}
            <Badge tone={book.source === 'audible' ? 'neutral' : 'info'} title="Metadata source">
              {metadataSourceMeta[book.source].label}
            </Badge>
          </h3>
          {book.subtitle && <div className="text-xs text-muted">{book.subtitle}</div>}
          <div className="mt-1 text-sm text-muted">
            {book.authors.length > 0 ? book.authors.map((a) => a.name).join(', ') : 'Unknown author'}
          </div>
          {book.narrators.length > 0 && (
            <div className="text-xs text-faint">Narrated by {book.narrators.join(', ')}</div>
          )}
          {facts.length > 0 && <div className="mt-1 text-xs text-muted">{facts.join(' · ')}</div>}
          {description && <p className="mt-2 line-clamp-2 text-xs text-faint">{description}</p>}
        </div>
        <div className="shrink-0">
          {libraryId !== null ? (
            <LinkButton to={`/books/${libraryId}`} tone="ok" icon={<Check className="size-4" />}>
              In library
            </LinkButton>
          ) : (
            <Button
              icon={<Plus className="size-4" />}
              onClick={() => setOptionsOpen(!optionsOpen)}
              aria-expanded={optionsOpen}
            >
              Add
            </Button>
          )}
        </div>
      </div>
      {optionsOpen && libraryId === null && (
        <div className="mt-3 flex flex-wrap items-center gap-x-6 gap-y-3 rounded-control bg-surface-2 px-4 py-3">
          <Checkbox checked={monitored} onChange={setMonitored} label="Monitored" />
          <Checkbox checked={searchOnAdd} onChange={setSearchOnAdd} label="Search for the book now" />
          <div className="ml-auto flex gap-2">
            <Button size="sm" variant="ghost" onClick={() => setOptionsOpen(false)} disabled={add.isPending}>
              Cancel
            </Button>
            <Button
              size="sm"
              variant="primary"
              loading={add.isPending}
              onClick={() =>
                add.mutate(
                  { asin: book.asin, monitored, searchOnAdd },
                  { onSuccess: (b) => setAdded(b) },
                )
              }
            >
              Add book
            </Button>
          </div>
        </div>
      )}
    </li>
  )
}
