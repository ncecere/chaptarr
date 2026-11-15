import { useQuery } from '@tanstack/react-query'
import { CircleCheck, Download, ExternalLink, RefreshCw, TriangleAlert } from 'lucide-react'
import { useEffect, useState } from 'react'
import { api, type Release } from '../lib/api'
import { cn } from '../lib/cn'
import { formatAge, formatBytes } from '../lib/format'
import { useGrabRelease } from '../lib/mutations'
import { Badge } from './Badge'
import { Button } from './Button'
import { Checkbox } from './Checkbox'
import { Modal } from './Modal'
import { ErrorState } from './States'

function Elapsed() {
  const [secs, setSecs] = useState(0)
  useEffect(() => {
    const t = setInterval(() => setSecs((s) => s + 1), 1000)
    return () => clearInterval(t)
  }, [])
  return <span className="tabular-nums">{secs}s</span>
}

function SearchProgress({ label }: { label: string }) {
  return (
    <div role="status" className="flex flex-col items-center gap-3 py-14 text-sm text-muted">
      <div className="relative h-1 w-64 max-w-full overflow-hidden rounded-full bg-surface-3">
        <div className="absolute inset-y-0 left-0 w-1/3 rounded-full bg-accent animate-indeterminate" />
      </div>
      <div>
        {label} <Elapsed />
      </div>
      <div className="text-xs text-faint">Querying every enabled indexer. This can take 10 seconds or more.</div>
    </div>
  )
}

function GrabButton({ bookId, release, best }: { bookId: number; release: Release; best?: boolean }) {
  const m = useGrabRelease()
  if (m.isSuccess) {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-ok">
        <CircleCheck className="size-4" />
        Grabbed
      </span>
    )
  }
  const rejected = release.rejections.length > 0
  return (
    <Button
      size="sm"
      variant={best && !rejected ? 'primary' : 'secondary'}
      loading={m.isPending}
      icon={<Download className="size-3.5" />}
      title={rejected ? 'Automatic search would skip this release' : undefined}
      onClick={() => m.mutate({ bookId, releaseId: release.id })}
    >
      Grab
    </Button>
  )
}

const th = 'th'

export function ReleaseSearchModal({
  bookId,
  bookTitle,
  open,
  onClose,
}: {
  bookId: number
  bookTitle: string
  open: boolean
  onClose: () => void
}) {
  const [hideRejected, setHideRejected] = useState(false)
  // Indexer searches are slow and ids stay valid for ~30 minutes, so cache the result
  // and only re-run on request.
  const q = useQuery({
    queryKey: ['releases', bookId],
    queryFn: () => api.bookReleases(bookId),
    enabled: open,
    staleTime: 10 * 60_000,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retry: false,
  })

  const releases = q.data ?? []
  const shown = hideRejected ? releases.filter((r) => r.rejections.length === 0) : releases

  return (
    <Modal
      open={open}
      onClose={onClose}
      size="xl"
      title={`Interactive search — ${bookTitle}`}
      footer={
        <>
          {q.data && (
            <span className="mr-auto text-xs text-muted">
              {shown.length} of {releases.length} releases
            </span>
          )}
          <Button
            icon={<RefreshCw className={cn('size-4', q.isFetching && 'animate-spin')} />}
            disabled={q.isFetching}
            onClick={() => void q.refetch()}
          >
            Search again
          </Button>
          <Button variant="ghost" onClick={onClose}>
            Close
          </Button>
        </>
      }
    >
      {q.isFetching ? (
        <SearchProgress label={q.data ? 'Searching again…' : 'Searching indexers…'} />
      ) : q.isError ? (
        <ErrorState title="Search failed" error={q.error} onRetry={() => void q.refetch()} />
      ) : q.data && releases.length === 0 ? (
        <div className="py-14 text-center text-sm text-muted">
          No releases found. Check that your indexers are enabled and cover audiobooks.
        </div>
      ) : q.data ? (
        <div className="space-y-3">
          <Checkbox
            checked={hideRejected}
            onChange={setHideRejected}
            label="Hide releases automatic search would skip"
            className="text-muted"
          />
          <ul className="divide-y divide-line overflow-hidden rounded-card border border-line sm:hidden">
            {shown.map((r, i) => (
              <ReleaseCard key={r.id} bookId={bookId} release={r} best={i === 0} />
            ))}
            {shown.length === 0 && (
              <li className="px-3 py-8 text-center text-sm text-muted">
                Every release was rejected. Untick the filter to grab one manually.
              </li>
            )}
          </ul>
          <div className="hidden overflow-x-auto rounded-card border border-line sm:block">
            <table className="w-full min-w-[56rem] border-collapse text-sm">
              <thead>
                <tr className="border-b border-line bg-surface-2/50">
                  <th className={th}>Release</th>
                  <th className={th}>Indexer</th>
                  <th className={th}>Protocol</th>
                  <th className={th}>Format</th>
                  <th className={th}>Size</th>
                  <th className={th} title="Seeders / leechers">
                    Peers
                  </th>
                  <th className={th}>Age</th>
                  <th className={th}>Score</th>
                  <th className="w-24"><span className="sr-only">Grab</span></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {shown.map((r, i) => (
                  <ReleaseRow key={r.id} bookId={bookId} release={r} best={i === 0} />
                ))}
              </tbody>
            </table>
            {shown.length === 0 && (
              <div className="py-8 text-center text-sm text-muted">
                Every release was rejected. Untick the filter to grab one manually.
              </div>
            )}
          </div>
        </div>
      ) : null}
    </Modal>
  )
}

/** One release on a phone: the table's columns folded into a title and a line of facts. */
function ReleaseCard({ bookId, release: r, best }: { bookId: number; release: Release; best?: boolean }) {
  const rejected = r.rejections.length > 0
  const facts = [
    r.indexer,
    r.format && r.format.toUpperCase(),
    r.sizeBytes > 0 && formatBytes(r.sizeBytes),
    formatAge(r.ageDays),
    r.seeders !== null && `${r.seeders} / ${r.leechers ?? 0} peers`,
  ].filter(Boolean)
  return (
    <li className={cn('flex items-start gap-3 px-3 py-2.5', rejected && 'bg-warn/4')}>
      <div className="min-w-0 flex-1">
        <div className="break-words text-sm">{r.title}</div>
        <div className="mt-0.5 text-xs text-muted">{facts.join(' · ')}</div>
        {rejected && (
          <div className="mt-1 flex gap-1.5 text-xs text-warn">
            <TriangleAlert className="mt-px size-3.5 shrink-0" aria-label="Rejected" />
            <span>{r.rejections.join(' · ')}</span>
          </div>
        )}
      </div>
      <GrabButton bookId={bookId} release={r} best={best} />
    </li>
  )
}

function ReleaseRow({ bookId, release: r, best }: { bookId: number; release: Release; best?: boolean }) {
  const rejected = r.rejections.length > 0
  return (
    <tr className={cn('align-top transition-colors hover:bg-surface-2/60', rejected && 'bg-warn/4')}>
      <td className="max-w-md px-3 py-2">
        <div className="flex items-start gap-1.5">
          {rejected && (
            <span title={r.rejections.join('\n')} className="mt-0.5 shrink-0 text-warn">
              <TriangleAlert className="size-4" aria-label="Rejected" />
            </span>
          )}
          <span className="min-w-0 break-words">{r.title}</span>
          {r.infoUrl && (
            <a
              href={r.infoUrl}
              target="_blank"
              rel="noreferrer noopener"
              aria-label="Open release page"
              className="mt-0.5 shrink-0 text-faint hover:text-fg"
            >
              <ExternalLink className="size-3.5" />
            </a>
          )}
        </div>
        {rejected && <div className="mt-1 pl-5.5 text-xs text-warn">{r.rejections.join(' · ')}</div>}
      </td>
      <td className="px-3 py-2 whitespace-nowrap text-muted">{r.indexer}</td>
      <td className="px-3 py-2">
        <Badge tone={r.protocol === 'torrent' ? 'info' : 'neutral'}>
          {r.protocol === 'torrent' ? 'Torrent' : 'Usenet'}
        </Badge>
      </td>
      <td className="px-3 py-2 whitespace-nowrap text-muted uppercase">{r.format || '—'}</td>
      <td className="px-3 py-2 whitespace-nowrap text-muted tabular-nums">
        {r.sizeBytes > 0 ? formatBytes(r.sizeBytes) : '—'}
      </td>
      <td className="px-3 py-2 whitespace-nowrap tabular-nums">
        {r.seeders === null ? (
          <span className="text-faint">—</span>
        ) : (
          <>
            <span className={r.seeders === 0 ? 'text-bad' : 'text-ok'}>{r.seeders}</span>
            <span className="text-faint"> / {r.leechers ?? 0}</span>
          </>
        )}
      </td>
      <td className="px-3 py-2 whitespace-nowrap text-muted tabular-nums">{formatAge(r.ageDays)}</td>
      <td className="px-3 py-2 whitespace-nowrap text-muted tabular-nums">{r.score}</td>
      <td className="px-3 py-2 text-right">
        <GrabButton bookId={bookId} release={r} best={best} />
      </td>
    </tr>
  )
}
