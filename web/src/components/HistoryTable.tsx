import { Link } from 'react-router-dom'
import type { HistoryEvent } from '../lib/api'
import { cn } from '../lib/cn'
import { formatDateTime, formatRelative } from '../lib/format'
import { HistoryTypeBadge } from './StatusBadge'

const th = 'th'

export function HistoryTable({ events, showBook }: { events: HistoryEvent[]; showBook?: boolean }) {
  return (
    <div className="overflow-x-auto rounded-card border border-line bg-surface">
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
                {e.message && (
                  <div
                    className={cn(
                      'break-words text-xs',
                      e.type === 'downloadFailed' || e.type === 'importFailed' ? 'text-bad' : 'text-muted',
                    )}
                  >
                    {e.message}
                  </div>
                )}
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
  )
}
