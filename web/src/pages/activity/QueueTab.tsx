import { CircleAlert, Download, FolderInput, RotateCw, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Badge } from '../../components/Badge'
import { Button, IconButton } from '../../components/Button'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { Cover } from '../../components/Cover'
import { ManualImportModal } from '../../components/ManualImportModal'
import { ProgressBar } from '../../components/ProgressBar'
import { QueryView } from '../../components/QueryView'
import { EmptyState } from '../../components/States'
import { QueueStatusBadge } from '../../components/StatusBadge'
import type { QueueItem } from '../../lib/api'
import { formatBytes, formatEta, formatPercent } from '../../lib/format'
import { useRemoveQueueItem, useRetryImport } from '../../lib/mutations'
import { useQueue } from '../../lib/queries'

function QueueRow({ item, onRemove }: { item: QueueItem; onRemove: () => void }) {
  const retry = useRetryImport()
  const [manual, setManual] = useState(false)
  const canRetry = item.status === 'completed' || item.status === 'failed'
  const progress = item.status === 'completed' || item.status === 'importing' ? 1 : item.progress
  const tone = item.status === 'failed' ? 'bad' : item.status === 'paused' ? 'warn' : progress >= 1 ? 'ok' : 'accent'

  return (
    <li className="flex flex-wrap gap-3 p-3 sm:flex-nowrap sm:gap-4 sm:p-4">
      <Cover src={item.coverUrl} alt="" className="size-14 sm:size-16" />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-baseline gap-x-2">
          <Link to={`/books/${item.bookId}`} className="font-medium transition-colors hover:text-accent">
            {item.bookTitle}
          </Link>
          <span className="text-xs text-muted">{item.authorName}</span>
        </div>
        <div className="mt-0.5 truncate text-xs text-faint" title={item.releaseTitle}>
          {item.releaseTitle}
        </div>
        <div className="mt-2 flex flex-wrap items-center gap-x-2.5 gap-y-1 text-xs text-muted">
          <QueueStatusBadge status={item.status} />
          <Badge tone={item.protocol === 'torrent' ? 'info' : 'neutral'}>
            {item.protocol === 'torrent' ? 'Torrent' : 'Usenet'}
          </Badge>
          <span>{item.downloadClient}</span>
          {item.indexer && <span className="text-faint">via {item.indexer}</span>}
          {item.sizeBytes > 0 && <span>{formatBytes(item.sizeBytes)}</span>}
        </div>
        <div className="mt-2.5 flex items-center gap-3">
          <ProgressBar label="Download progress" value={progress} tone={tone} className="max-w-md flex-1" />
          <span className="w-9 text-right text-xs text-muted tabular-nums">{formatPercent(progress)}</span>
          {item.status === 'downloading' && (
            <span className="text-xs text-muted tabular-nums" title="Estimated time remaining">
              ETA {formatEta(item.etaSeconds)}
            </span>
          )}
        </div>
        {item.error && (
          <p className="mt-2.5 flex max-w-2xl gap-2 rounded-control bg-bad/8 px-2.5 py-2 text-xs leading-relaxed text-muted">
            <CircleAlert className="mt-px size-3.5 shrink-0 text-bad" />
            <span className="min-w-0 break-words">{item.error}</span>
          </p>
        )}
      </div>
      <div className="flex shrink-0 basis-full items-center justify-end gap-1.5 sm:basis-auto sm:items-start">
        {canRetry && (
          <Button
            size="sm"
            icon={<RotateCw className="size-3.5" />}
            loading={retry.isPending}
            onClick={() => retry.mutate(item.id)}
          >
            Retry Import
          </Button>
        )}
        <Button size="sm" icon={<FolderInput className="size-3.5" />} onClick={() => setManual(true)}>
          Manual Import
        </Button>
        <IconButton
          size="sm"
          label={`Remove ${item.bookTitle} from the queue`}
          className="hover:bg-bad/10 hover:text-bad"
          onClick={onRemove}
        >
          <Trash2 className="size-4" />
        </IconButton>
      </div>
      {manual && (
        <ManualImportModal
          book={{ id: item.bookId, title: item.bookTitle }}
          queueId={item.id}
          onClose={() => setManual(false)}
        />
      )}
    </li>
  )
}

export function QueueTab() {
  const query = useQueue()
  const remove = useRemoveQueueItem()
  const [removing, setRemoving] = useState<QueueItem | null>(null)

  return (
    <>
      <QueryView
        query={query}
        isEmpty={(q) => q.length === 0}
        empty={
          <EmptyState
            icon={<Download />}
            title="Nothing in the queue"
            description="Releases you grab will show up here while they download and import."
          />
        }
      >
        {(items) => (
          <ul className="divide-y divide-line overflow-hidden rounded-card border border-line bg-surface">
            {items.map((it) => (
              <QueueRow key={it.id} item={it} onRemove={() => setRemoving(it)} />
            ))}
          </ul>
        )}
      </QueryView>

      <ConfirmDialog
        open={removing !== null}
        title="Remove from queue?"
        danger
        confirmLabel="Remove"
        message={
          <>
            Remove <span className="font-medium text-fg">{removing?.bookTitle}</span> from the queue.
          </>
        }
        checkbox={{ label: 'Also remove it from the download client (deletes the download)' }}
        pending={remove.isPending}
        onCancel={() => setRemoving(null)}
        onConfirm={(removeFromClient) => {
          if (!removing) return
          remove.mutate(
            { id: removing.id, removeFromClient },
            { onSuccess: () => setRemoving(null) },
          )
        }}
      />
    </>
  )
}
