import { CircleAlert } from 'lucide-react'
import { Link } from 'react-router-dom'
import type { HistoryEvent } from '../lib/api'
import { cn } from '../lib/cn'
import { formatDateTime, formatRelative } from '../lib/format'
import { HistoryTypeBadge } from './StatusBadge'

const th = 'th'

export function HistoryTable({ events, showBook }: { events: HistoryEvent[]; showBook?: boolean }) {
  return (
    <div className="overflow-hidden rounded-card border border-line bg-surface">
      <ul className="divide-y divide-line sm:hidden">
        {events.map((e) => (
          <li key={e.id} className="space-y-1 px-3 py-2.5">
            <div className="flex items-center justify-between gap-3">
              <HistoryTypeBadge type={e.type} />
              <span className="text-xs text-muted" title={formatDateTime(e.date)}>
                {formatRelative(e.date)}
              </span>
            </div>
            {showBook && (
              <Link to={`/books/${e.bookId}`} className="block truncate text-sm font-medium">
                {e.bookTitle}
              </Link>
            )}
            {e.sourceTitle && <div className="break-words text-xs text-muted">{e.sourceTitle}</div>}
            {e.message && <HistoryMessage event={e} />}
          </li>
        ))}
      </ul>
      <div className="hidden overflow-x-auto sm:block">
      <table className="w-full min-w-[36rem] border-collapse text-sm">
        <thead>
          <tr className="border-b border-line bg-surface-2/50">
            <th className={th}>Event</th>
            {showBook && <th className={th}>Book</th>}
            <th className={th}>Details</th>
            <th className={cn(th, 'text-right')}>Date</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line">
          {events.map((e) => (
            <tr key={e.id} className="align-top transition-colors hover:bg-surface-2/60">
              <td className="px-3 py-2">
                <HistoryTypeBadge type={e.type} />
              </td>
              {showBook && (
                <td className="max-w-64 px-3 py-2">
                  <Link to={`/books/${e.bookId}`} className="block truncate font-medium hover:text-accent">
                    {e.bookTitle}
                  </Link>
                  <div className="truncate text-xs text-muted">{e.authorName}</div>
                </td>
              )}
              <td className="max-w-md px-3 py-2">
                {e.sourceTitle && <div className="break-words">{e.sourceTitle}</div>}
                {e.message && <HistoryMessage event={e} />}
              </td>
              <td
                className="px-3 py-2 text-right whitespace-nowrap text-muted"
                title={formatDateTime(e.date)}
              >
                {formatRelative(e.date)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      </div>
    </div>
  )
}

/** The event's note. A failure gets a warning mark; the text stays readable rather than red. */
function HistoryMessage({ event: e }: { event: HistoryEvent }) {
  const failed = e.type === 'downloadFailed' || e.type === 'importFailed'
  return (
    <div className="flex gap-1.5 text-xs leading-relaxed text-muted">
      {failed && <CircleAlert className="mt-0.5 size-3.5 shrink-0 text-bad" aria-label="Failed" />}
      <span className="min-w-0 break-words">{e.message}</span>
    </div>
  )
}
