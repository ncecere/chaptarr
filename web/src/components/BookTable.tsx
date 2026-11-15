import { Link } from 'react-router-dom'
import type { Book } from '../lib/api'
import { cn } from '../lib/cn'
import { formatBytes, formatDate, formatRuntime } from '../lib/format'
import { MonitorToggle, SearchBookButton } from './BookActions'
import { Checkbox } from './Checkbox'
import { Cover } from './Cover'
import { BookStatusBadge } from './StatusBadge'

export interface BookSelection {
  selected: ReadonlySet<number>
  onToggle: (id: number) => void
  onToggleMany: (ids: number[], on: boolean) => void
}

interface BookTableProps {
  books: Book[]
  showCover?: boolean
  showAuthor?: boolean
  showSeries?: boolean
  /** Series position column; for lists already scoped to one series. */
  showPosition?: boolean
  showSize?: boolean
  showMonitor?: boolean
  showSearch?: boolean
  selection?: BookSelection
  className?: string
}

const th = 'th'

export function BookTable({
  books,
  showCover,
  showAuthor,
  showSeries,
  showPosition,
  showSize,
  showMonitor,
  showSearch,
  selection,
  className,
}: BookTableProps) {
  const ids = books.map((b) => b.id)
  const selectedCount = selection ? ids.filter((id) => selection.selected.has(id)).length : 0
  return (
    <div className={cn('overflow-hidden rounded-card border border-line bg-surface', className)}>
      {/* Phones: one stacked row per book instead of a table that scrolls sideways. */}
      <div className="sm:hidden">
        {selection && (
          <div className="flex items-center gap-3 border-b border-line bg-surface-2/50 px-3 py-2">
            <Checkbox
              checked={books.length > 0 && selectedCount === books.length}
              indeterminate={selectedCount > 0 && selectedCount < books.length}
              onChange={(on) => selection.onToggleMany(ids, on)}
              label={<span className="text-xs text-muted">Select all</span>}
            />
          </div>
        )}
        <ul className="divide-y divide-line">
          {books.map((b) => {
            const meta = [
              showPosition && b.series?.position && `#${b.series.position}`,
              showAuthor && b.authorName,
              !showPosition && showSeries && b.series && `${b.series.name}${b.series.position ? ` #${b.series.position}` : ''}`,
              b.runtimeMinutes > 0 && formatRuntime(b.runtimeMinutes),
            ].filter(Boolean)
            return (
              <li key={b.id} className="flex items-center gap-3 px-3 py-2.5">
                {selection && (
                  <Checkbox
                    aria-label={`Select ${b.title}`}
                    checked={selection.selected.has(b.id)}
                    onChange={() => selection.onToggle(b.id)}
                  />
                )}
                {showCover && <Cover src={b.coverUrl} alt="" className="size-11 rounded-chip" />}
                <div className="min-w-0 flex-1">
                  <Link to={`/books/${b.id}`} className="block truncate text-sm font-medium" title={b.title}>
                    {b.title}
                  </Link>
                  {meta.length > 0 && <div className="truncate text-xs text-muted">{meta.join(' · ')}</div>}
                  <BookStatusBadge status={b.status} dot className="mt-0.5 text-xs" />
                </div>
                {showMonitor && <MonitorToggle book={b} />}
                {showSearch && <SearchBookButton bookId={b.id} title={b.title} />}
              </li>
            )
          })}
        </ul>
      </div>
      <div className="hidden overflow-x-auto sm:block">
      <table className="w-full min-w-[34rem] border-collapse text-sm">
        <thead>
          <tr className="border-b border-line bg-surface-2/50">
            {selection && (
              <th className="w-10 py-2 pr-0 pl-3">
                <Checkbox
                  aria-label="Select all"
                  checked={books.length > 0 && selectedCount === books.length}
                  indeterminate={selectedCount > 0 && selectedCount < books.length}
                  onChange={(on) => selection.onToggleMany(ids, on)}
                />
              </th>
            )}
            {showCover && <th className="w-14"><span className="sr-only">Cover</span></th>}
            {showPosition && <th className={cn(th, 'w-14')}>#</th>}
            <th className={th}>Title</th>
            {showAuthor && <th className={cn(th, 'hidden lg:table-cell')}>Author</th>}
            {showSeries && <th className={cn(th, 'hidden xl:table-cell')}>Series</th>}
            <th className={cn(th, 'hidden md:table-cell')}>Released</th>
            <th className={cn(th, 'hidden md:table-cell')}>Runtime</th>
            {showSize && <th className={cn(th, 'hidden lg:table-cell')}>Size</th>}
            <th className={th}>Status</th>
            {showMonitor && <th className={cn(th, 'w-20')}>Monitor</th>}
            {showSearch && <th className="w-12"><span className="sr-only">Search</span></th>}
          </tr>
        </thead>
        <tbody className="divide-y divide-line">
          {books.map((b) => (
            <tr key={b.id} className="transition-colors hover:bg-surface-2/60">
              {selection && (
                <td className="py-2 pr-0 pl-3">
                  <Checkbox
                    aria-label={`Select ${b.title}`}
                    checked={selection.selected.has(b.id)}
                    onChange={() => selection.onToggle(b.id)}
                  />
                </td>
              )}
              {showCover && (
                <td className="py-2 pl-3">
                  <Cover src={b.coverUrl} alt="" className="size-9 rounded-chip" />
                </td>
              )}
              {showPosition && (
                <td className="px-3 py-2 text-faint tabular-nums">{b.series?.position || '—'}</td>
              )}
              <td className="w-full max-w-0 min-w-48 px-3 py-2">
                <Link to={`/books/${b.id}`} className="block truncate font-medium transition-colors hover:text-accent" title={b.title}>
                  {b.title}
                </Link>
                {(b.subtitle || showAuthor) && (
                  <div className="truncate text-xs text-muted">
                    {b.subtitle}
                    {showAuthor && (
                      <span className="lg:hidden">
                        {b.subtitle ? ' · ' : ''}
                        {b.authorName}
                      </span>
                    )}
                  </div>
                )}
              </td>
              {showAuthor && (
                <td className="hidden max-w-44 truncate whitespace-nowrap px-3 py-2 text-muted lg:table-cell">
                  <Link to={`/authors/${b.authorId}`} className="hover:text-fg">
                    {b.authorName}
                  </Link>
                </td>
              )}
              {showSeries && (
                <td className="hidden max-w-44 truncate whitespace-nowrap px-3 py-2 text-muted xl:table-cell">
                  {b.series ? (
                    <Link to={`/series/${b.series.id}`} className="hover:text-fg">
                      {b.series.name}
                      {b.series.position && ` #${b.series.position}`}
                    </Link>
                  ) : (
                    '—'
                  )}
                </td>
              )}
              <td className="hidden whitespace-nowrap px-3 py-2 text-muted tabular-nums md:table-cell">
                {formatDate(b.releaseDate)}
              </td>
              <td className="hidden whitespace-nowrap px-3 py-2 text-muted tabular-nums md:table-cell">
                {formatRuntime(b.runtimeMinutes)}
              </td>
              {showSize && (
                <td className="hidden whitespace-nowrap px-3 py-2 text-muted tabular-nums lg:table-cell">
                  {b.sizeBytes > 0 ? formatBytes(b.sizeBytes) : '—'}
                  {b.format && <span className="ml-1.5 text-2xs font-medium uppercase tracking-wide text-faint">{b.format}</span>}
                </td>
              )}
              <td className="whitespace-nowrap px-3 py-2">
                <BookStatusBadge status={b.status} dot />
              </td>
              {showMonitor && (
                <td className="px-3 py-2">
                  <MonitorToggle book={b} />
                </td>
              )}
              {showSearch && (
                <td className="py-2 pr-3 text-right">
                  <SearchBookButton bookId={b.id} title={b.title} />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      </div>
    </div>
  )
}
