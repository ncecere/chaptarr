import { ListPlus, RefreshCw, Search, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { BookTable, type BookSelection } from '../components/BookTable'
import { Button } from '../components/Button'
import { CollapsibleText } from '../components/CollapsibleText'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { Cover } from '../components/Cover'
import { ImportCatalogueModal } from '../components/ImportCatalogueModal'
import { PageHeader } from '../components/PageHeader'
import { ProgressBar } from '../components/ProgressBar'
import { QueryView } from '../components/QueryView'
import { EmptyState } from '../components/States'
import { LabeledToggle } from '../components/Toggle'
import type { AuthorDetail, Book, SeriesRef } from '../lib/api'
import { plural } from '../lib/format'
import { useDocumentTitle } from '../lib/hooks'
import {
  useDeleteAuthor,
  useRefreshAuthor,
  useSearchAuthor,
  useSetBooksMonitored,
  useUpdateAuthor,
} from '../lib/mutations'
import { useAuthor } from '../lib/queries'
import { seriesPositionValue } from '../lib/status'
import { htmlToText } from '../lib/text'

interface Group {
  key: string
  series: SeriesRef | null
  books: Book[]
}

/** Series groups (A–Z, books in position order) followed by standalones, newest first. */
function groupBooks(books: Book[]): Group[] {
  const bySeries = new Map<number, Group>()
  const standalone: Book[] = []
  for (const b of books) {
    if (!b.series) {
      standalone.push(b)
      continue
    }
    const g = bySeries.get(b.series.id)
    if (g) g.books.push(b)
    else bySeries.set(b.series.id, { key: `s${b.series.id}`, series: b.series, books: [b] })
  }
  const groups = [...bySeries.values()].sort((a, b) => a.series!.name.localeCompare(b.series!.name))
  for (const g of groups) {
    g.books.sort(
      (a, b) =>
        seriesPositionValue(a.series?.position) - seriesPositionValue(b.series?.position) ||
        a.releaseDate.localeCompare(b.releaseDate),
    )
  }
  if (standalone.length > 0) {
    standalone.sort((a, b) => b.releaseDate.localeCompare(a.releaseDate) || a.title.localeCompare(b.title))
    groups.push({ key: 'standalone', series: null, books: standalone })
  }
  return groups
}

function AuthorBody({ author }: { author: AuthorDetail }) {
  useDocumentTitle(author.name)
  const navigate = useNavigate()
  const update = useUpdateAuthor()
  const refresh = useRefreshAuthor()
  const searchMissing = useSearchAuthor()
  const del = useDeleteAuthor()
  const setMonitored = useSetBooksMonitored()
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [importOpen, setImportOpen] = useState(false)
  const [selected, setSelected] = useState<ReadonlySet<number>>(new Set())

  const groups = useMemo(() => groupBooks(author.books), [author.books])
  const bio = useMemo(() => htmlToText(author.description), [author.description])

  // Show the requested value while a toggle request is in flight.
  const pending = update.isPending ? update.variables.patch : undefined
  const monitored = pending?.monitored ?? author.monitored
  const monitorNew = pending?.monitorNewBooks ?? author.monitorNewBooks

  const selection: BookSelection = {
    selected,
    onToggle: (id) => {
      const next = new Set(selected)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      setSelected(next)
    },
    onToggleMany: (ids, on) => {
      const next = new Set(selected)
      for (const id of ids) {
        if (on) next.add(id)
        else next.delete(id)
      }
      setSelected(next)
    },
  }

  const bulk = (on: boolean) =>
    setMonitored.mutate(
      { ids: [...selected], monitored: on },
      { onSuccess: () => setSelected(new Set()) },
    )

  return (
    <>
      <PageHeader back={{ to: '/authors', label: 'All authors' }} title={author.name} />

      <section className="mb-8 flex flex-col gap-5 rounded-card border border-line bg-surface p-4 sm:flex-row sm:p-5">
        <Cover src={author.imageUrl} alt={author.name} kind="author" className="hidden size-36 sm:block" />
        <div className="min-w-0 flex-1 space-y-4">
          {/* On a phone the photo shrinks and sits beside the progress, so the page starts with content. */}
          <div className="flex items-center gap-4">
            <Cover src={author.imageUrl} alt="" kind="author" className="size-16 sm:hidden" />
            <div className="min-w-0 max-w-md flex-1">
              <ProgressBar label="Books downloaded"
                value={author.bookCount > 0 ? author.downloadedCount / author.bookCount : 0}
                tone={author.bookCount > 0 && author.downloadedCount === author.bookCount ? 'ok' : 'accent'}
              />
              <div className="mt-1 flex justify-between text-xs text-muted">
                <span>
                  {author.downloadedCount} of {plural(author.bookCount, 'book')} downloaded
                </span>
                {author.missingCount > 0 && <span className="text-warn">{author.missingCount} missing</span>}
              </div>
            </div>
          </div>
          {bio && <CollapsibleText text={bio} className="max-w-3xl" />}
          <div className="flex flex-wrap gap-x-8 gap-y-3">
            <LabeledToggle
              label="Monitored"
              checked={monitored}
              onChange={(v) => update.mutate({ id: author.id, patch: { monitored: v } })}
            />
            <LabeledToggle
              label="Monitor new books"
              hint="Add and monitor future releases"
              checked={monitorNew}
              onChange={(v) => update.mutate({ id: author.id, patch: { monitorNewBooks: v } })}
            />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button icon={<ListPlus className="size-4" />} onClick={() => setImportOpen(true)}>
              Add All Books
            </Button>
            <Button
              icon={<RefreshCw className="size-4" />}
              loading={refresh.isPending}
              onClick={() => refresh.mutate(author.id)}
            >
              Refresh
            </Button>
            <Button
              icon={<Search className="size-4" />}
              loading={searchMissing.isPending}
              onClick={() => searchMissing.mutate(author.id)}
            >
              Search Missing
            </Button>
            <Button variant="danger" icon={<Trash2 className="size-4" />} onClick={() => setConfirmDelete(true)}>
              Delete
            </Button>
          </div>
        </div>
      </section>

      {selected.size > 0 && (
        <div className="sticky top-14 z-20 mb-4 flex flex-wrap items-center gap-2 rounded-card border border-line-strong bg-surface py-2 pr-2 pl-4 shadow-overlay lg:top-2">
          <span className="mr-auto text-sm font-medium">{selected.size} selected</span>
          <Button size="sm" loading={setMonitored.isPending} onClick={() => bulk(true)}>
            Monitor
          </Button>
          <Button size="sm" loading={setMonitored.isPending} onClick={() => bulk(false)}>
            Unmonitor
          </Button>
          <Button size="sm" variant="ghost" onClick={() => setSelected(new Set())}>
            Clear
          </Button>
        </div>
      )}

      {author.books.length === 0 ? (
        <EmptyState
          title="No books yet"
          description="Refresh the author to pull their catalogue from the metadata provider."
        />
      ) : (
        <div className="space-y-8">
          {groups.map((g) => (
            <section key={g.key}>
              <h2 className="mb-2 flex items-baseline gap-2 text-base font-semibold">
                {g.series ? (
                  <Link to={`/series/${g.series.id}`} className="hover:text-accent">
                    {g.series.name}
                  </Link>
                ) : (
                  'Standalone'
                )}
                <span className="text-xs font-normal text-muted">{plural(g.books.length, 'book')}</span>
              </h2>
              <BookTable
                books={g.books}
                showPosition={!!g.series}
                showMonitor
                showSearch
                selection={selection}
              />
            </section>
          ))}
        </div>
      )}

      {importOpen && <ImportCatalogueModal author={author} open onClose={() => setImportOpen(false)} />}

      <ConfirmDialog
        open={confirmDelete}
        title={`Delete ${author.name}?`}
        danger
        confirmLabel="Delete"
        message={`This removes the author and ${plural(author.bookCount, 'book')} from Chaptarr.`}
        checkbox={{ label: 'Also delete the files from disk' }}
        pending={del.isPending}
        onCancel={() => setConfirmDelete(false)}
        onConfirm={(deleteFiles) =>
          del.mutate({ id: author.id, deleteFiles }, { onSuccess: () => navigate('/authors') })
        }
      />
    </>
  )
}

export function AuthorDetailPage() {
  const id = Number(useParams().id)
  const query = useAuthor(id)
  return <QueryView query={query}>{(author) => <AuthorBody author={author} />}</QueryView>
}
