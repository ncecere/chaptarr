import type { BookStatus, HistoryEventType, QueueStatus } from '../lib/api'
import { bookStatusMeta, historyTypeMeta, queueStatusMeta } from '../lib/status'
import { Badge } from './Badge'

export function BookStatusBadge({ status, className }: { status: BookStatus; className?: string }) {
  const m = bookStatusMeta[status]
  return (
    <Badge tone={m.tone} className={className}>
      {m.label}
    </Badge>
  )
}

export function QueueStatusBadge({ status }: { status: QueueStatus }) {
  const m = queueStatusMeta[status]
  return <Badge tone={m.tone}>{m.label}</Badge>
}

export function HistoryTypeBadge({ type }: { type: HistoryEventType }) {
  const m = historyTypeMeta[type]
  return <Badge tone={m.tone}>{m.label}</Badge>
}
