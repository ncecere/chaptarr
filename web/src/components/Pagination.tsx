import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from './Button'

export function Pagination({
  page,
  pageSize,
  total,
  onPage,
  busy,
}: {
  page: number
  pageSize: number
  total: number
  onPage: (page: number) => void
  busy?: boolean
}) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  const from = total === 0 ? 0 : (page - 1) * pageSize + 1
  const to = Math.min(total, page * pageSize)
  return (
    <div className="mt-4 flex items-center justify-between gap-3 text-sm text-muted">
      <span>
        {from}–{to} of {total.toLocaleString()}
      </span>
      <div className="flex items-center gap-2">
        <Button
          size="sm"
          icon={<ChevronLeft className="size-3.5" />}
          disabled={page <= 1 || busy}
          onClick={() => onPage(page - 1)}
        >
          Prev
        </Button>
        <span className="tabular-nums">
          Page {page} of {pages}
        </span>
        <Button size="sm" disabled={page >= pages || busy} onClick={() => onPage(page + 1)}>
          Next
          <ChevronRight className="size-3.5" />
        </Button>
      </div>
    </div>
  )
}
