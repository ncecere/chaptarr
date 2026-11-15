import { ListPlus } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { AddSeriesBooksModal } from '../components/AddSeriesBooksModal'
import { BookTable } from '../components/BookTable'
import { Button } from '../components/Button'
import { Cover } from '../components/Cover'
import { PageHeader } from '../components/PageHeader'
import { ProgressBar } from '../components/ProgressBar'
import { QueryView } from '../components/QueryView'
import { EmptyState } from '../components/States'
import type { SeriesDetail } from '../lib/api'
import { plural } from '../lib/format'
import { useDocumentTitle } from '../lib/hooks'
import { useSeriesDetail } from '../lib/queries'
import { seriesPositionValue } from '../lib/status'

function SeriesBody({ series }: { series: SeriesDetail }) {
  const books = useMemo(
    () =>
      [...series.books].sort(
        (a, b) =>
          seriesPositionValue(a.series?.position) - seriesPositionValue(b.series?.position) ||
          a.releaseDate.localeCompare(b.releaseDate) ||
          a.title.localeCompare(b.title),
      ),
    [series.books],
  )
  useDocumentTitle(series.name)
  const [addOpen, setAddOpen] = useState(false)
  return (
    <>
      <PageHeader
        back={{ to: '/series', label: 'All series' }}
        title={series.name}
        description={`${plural(series.bookCount, 'book')} · ${series.downloadedCount} downloaded`}
      />
      <div className="mb-6 flex flex-wrap items-center gap-4 rounded-card border border-line bg-surface p-4">
        <Cover src={series.coverUrl} alt="" kind="series" className="size-16" />
        <div className="min-w-0 flex-1 text-sm">
          <div className="flex flex-wrap items-baseline justify-between gap-x-4 sm:max-w-xs">
            <span className="text-muted">
              by <span className="text-fg">{series.authorName || 'Unknown author'}</span>
            </span>
            <span className="text-xs text-muted tabular-nums">
              {series.downloadedCount} of {series.bookCount}
            </span>
          </div>
          <ProgressBar label="Books downloaded"
            className="mt-2 max-w-xs"
            value={series.bookCount > 0 ? series.downloadedCount / series.bookCount : 0}
            tone={series.bookCount > 0 && series.downloadedCount === series.bookCount ? 'ok' : 'accent'}
          />
        </div>
        <Button icon={<ListPlus className="size-4" />} onClick={() => setAddOpen(true)}>
          Add Missing Books
        </Button>
      </div>
      {addOpen && <AddSeriesBooksModal series={series} open onClose={() => setAddOpen(false)} />}
      {books.length === 0 ? (
        <EmptyState title="No books in this series" />
      ) : (
        <BookTable books={books} showCover showPosition showAuthor showMonitor showSearch />
      )}
    </>
  )
}

export function SeriesDetailPage() {
  const id = Number(useParams().id)
  const query = useSeriesDetail(id)
  return (
    <QueryView query={query}>
      {(series) => <SeriesBody series={series} />}
    </QueryView>
  )
}
