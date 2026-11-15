import { Layers, Search } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Cover } from '../components/Cover'
import { CoverGrid, CoverGridSkeleton } from '../components/CoverGrid'
import { Input } from '../components/Form'
import { PageHeader } from '../components/PageHeader'
import { ProgressBar } from '../components/ProgressBar'
import { QueryView } from '../components/QueryView'
import { EmptyState } from '../components/States'
import { plural } from '../lib/format'
import { useDocumentTitle } from '../lib/hooks'
import { useSeriesList } from '../lib/queries'

export function SeriesPage() {
  useDocumentTitle('Series')
  const [text, setText] = useState('')
  const query = useSeriesList()
  const list = useMemo(() => {
    if (!query.data) return undefined
    const needle = text.trim().toLowerCase()
    return query.data
      .filter(
        (s) =>
          !needle || s.name.toLowerCase().includes(needle) || s.authorName.toLowerCase().includes(needle),
      )
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [query.data, text])

  return (
    <>
      <PageHeader title="Series" description={query.data ? plural(query.data.length, 'series', 'series') : undefined} />
      <div className="relative mb-6 max-w-xs">
        <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
        <Input
          type="search"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Filter series"
          aria-label="Filter series"
          className="pl-9"
        />
      </div>
      <QueryView
        query={query}
        data={list}
        loading={<CoverGridSkeleton count={8} />}
        isEmpty={(s) => s.length === 0}
        empty={
          <EmptyState
            icon={<Layers />}
            title={query.data && query.data.length > 0 ? 'No series match' : 'No series yet'}
            description={
              query.data && query.data.length > 0
                ? 'Try a different name.'
                : 'Series appear here once books that belong to one are in your library.'
            }
          />
        }
      >
        {(items) => (
          <CoverGrid>
            {items.map((s) => (
              <Link key={s.id} to={`/series/${s.id}`} className="group block min-w-0">
                <Cover
                  src={s.coverUrl}
                  alt={`Cover of ${s.name}`}
                  kind="series"
                  className="transition-transform duration-200 group-hover:-translate-y-0.5 group-hover:ring-accent/50"
                />
                <div className="mt-2 min-w-0">
                  <div className="truncate text-sm font-medium group-hover:text-accent" title={s.name}>
                    {s.name}
                  </div>
                  <div className="truncate text-xs text-muted">{s.authorName}</div>
                  <ProgressBar
                    className="mt-1.5"
                    value={s.bookCount > 0 ? s.downloadedCount / s.bookCount : 0}
                    tone={s.bookCount > 0 && s.downloadedCount === s.bookCount ? 'ok' : 'accent'}
                  />
                  <div className="mt-0.5 text-xs text-muted tabular-nums">
                    {s.downloadedCount} / {s.bookCount} books
                  </div>
                </div>
              </Link>
            ))}
          </CoverGrid>
        )}
      </QueryView>
    </>
  )
}
