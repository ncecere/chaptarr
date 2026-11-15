import { useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { BookTable } from '../components/BookTable'
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
  return (
    <>
      <PageHeader
        back={{ to: '/series', label: 'All series' }}
        title={series.name}
        description={`${plural(series.bookCount, 'book')} · ${series.downloadedCount} downloaded`}
      />
      <div className="mb-6 flex items-center gap-4 rounded-xl border border-line bg-surface p-4">
        <Cover src={series.coverUrl} alt="" kind="series" className="size-16" />
        <div className="min-w-0 flex-1 text-sm">
          <div className="text-muted">
            by <span className="text-fg">{series.authorName || 'Unknown author'}</span>
          </div>
          <ProgressBar
            className="mt-2 max-w-xs"
            value={series.bookCount > 0 ? series.downloadedCount / series.bookCount : 0}
            tone={series.bookCount > 0 && series.downloadedCount === series.bookCount ? 'ok' : 'accent'}
          />
        </div>
      </div>
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
