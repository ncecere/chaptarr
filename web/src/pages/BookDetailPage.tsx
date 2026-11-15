import { FolderInput, Search, Trash2, Zap } from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { MonitorToggle } from '../components/BookActions'
import { Button } from '../components/Button'
import { CollapsibleText } from '../components/CollapsibleText'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { Cover } from '../components/Cover'
import { HistoryTable } from '../components/HistoryTable'
import { ManualImportModal } from '../components/ManualImportModal'
import { PageHeader } from '../components/PageHeader'
import { QueryView } from '../components/QueryView'
import { ReleaseSearchModal } from '../components/ReleaseSearchModal'
import { BookStatusBadge } from '../components/StatusBadge'
import type { BookDetail } from '../lib/api'
import { formatBytes, formatDate, formatRuntime } from '../lib/format'
import { useDocumentTitle } from '../lib/hooks'
import { metadataSourceMeta, sourceOfId } from '../lib/status'
import { useDeleteBook, useSearchBook } from '../lib/mutations'
import { useBook } from '../lib/queries'
import { htmlToText } from '../lib/text'

function Meta({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-[7rem_1fr] gap-3 py-1.5 text-sm">
      <dt className="text-muted">{label}</dt>
      <dd className="min-w-0 break-words">{children}</dd>
    </div>
  )
}

const th = 'th'

function BookBody({ book }: { book: BookDetail }) {
  useDocumentTitle(book.title)
  const source = metadataSourceMeta[sourceOfId(book.asin)]
  const navigate = useNavigate()
  const search = useSearchBook()
  const del = useDeleteBook()
  const [interactive, setInteractive] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [manualImport, setManualImport] = useState(false)
  const description = useMemo(() => htmlToText(book.description), [book.description])

  const authors = book.authors.length > 0 ? book.authors : [book.authorName]

  return (
    <>
      <PageHeader back={{ to: `/authors/${book.authorId}`, label: book.authorName }} title={book.title} description={book.subtitle} />

      <div className="grid gap-6 md:grid-cols-[15rem_1fr] lg:grid-cols-[18rem_1fr]">
        <div className="mx-auto w-full max-w-72 md:mx-0">
          <Cover src={book.coverUrl} alt={`Cover of ${book.title}`} className="w-full rounded-card shadow-overlay" />
        </div>

        <div className="min-w-0 space-y-6">
          <div className="flex flex-wrap items-center gap-x-5 gap-y-3">
            <BookStatusBadge status={book.status} />
            <div className="flex items-center gap-2 text-sm">
              <MonitorToggle book={book} size="md" />
              <span>{book.monitored ? 'Monitored' : 'Not monitored'}</span>
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            <Button
              variant="primary"
              icon={<Zap className="size-4" />}
              loading={search.isPending}
              onClick={() => search.mutate(book.id)}
            >
              Automatic Search
            </Button>
            <Button icon={<Search className="size-4" />} onClick={() => setInteractive(true)}>
              Interactive Search
            </Button>
            <Button icon={<FolderInput className="size-4" />} onClick={() => setManualImport(true)}>
              Manual Import
            </Button>
            <Button variant="danger" icon={<Trash2 className="size-4" />} onClick={() => setConfirmDelete(true)}>
              Delete
            </Button>
          </div>

          <dl className="divide-y divide-line rounded-card border border-line bg-surface px-4 py-1.5">
            <Meta label="Author">
              {authors.map((name, i) => (
                <span key={`${name}-${i}`}>
                  {i > 0 && ', '}
                  {i === 0 ? (
                    <Link to={`/authors/${book.authorId}`} className="text-accent hover:underline">
                      {name}
                    </Link>
                  ) : (
                    name
                  )}
                </span>
              ))}
            </Meta>
            {book.narrators.length > 0 && <Meta label="Narrated by">{book.narrators.join(', ')}</Meta>}
            {book.series && (
              <Meta label="Series">
                <Link to={`/series/${book.series.id}`} className="text-accent hover:underline">
                  {book.series.name}
                </Link>
                {book.series.position && <span className="text-muted"> · Book {book.series.position}</span>}
              </Meta>
            )}
            <Meta label="Runtime">{formatRuntime(book.runtimeMinutes)}</Meta>
            <Meta label="Released">{formatDate(book.releaseDate)}</Meta>
            {book.publisher && <Meta label="Publisher">{book.publisher}</Meta>}
            {book.language && <Meta label="Language">{book.language}</Meta>}
            <Meta label="Metadata">{source.label}</Meta>
            {book.asin && <Meta label={source.idLabel}>{book.asin.slice(source.prefix.length)}</Meta>}
            {book.path && (
              <Meta label="Location">
                <span className="font-mono text-xs">{book.path}</span>
              </Meta>
            )}
          </dl>

          {description && (
            <section>
              <h2 className="mb-1.5 text-sm font-semibold">About</h2>
              <CollapsibleText text={description} threshold={500} />
            </section>
          )}
        </div>
      </div>

      <section className="mt-10">
        <h2 className="mb-2 text-base font-semibold">Files</h2>
        {book.files.length === 0 ? (
          <p className="rounded-card border border-dashed border-line-strong px-4 py-6 text-center text-sm text-muted">
            No files on disk yet.
          </p>
        ) : (
          <div className="overflow-x-auto rounded-card border border-line bg-surface">
            <table className="w-full min-w-[28rem] border-collapse text-sm">
              <thead>
                <tr className="border-b border-line bg-surface-2/50">
                  <th className={th}>Path</th>
                  <th className={th}>Format</th>
                  <th className={th}>Size</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {book.files.map((f) => (
                  <tr key={f.id}>
                    <td className="px-3 py-2 font-mono text-xs break-all">{f.path}</td>
                    <td className="px-3 py-2 whitespace-nowrap uppercase text-muted">{f.format || '—'}</td>
                    <td className="px-3 py-2 whitespace-nowrap text-muted tabular-nums">{formatBytes(f.sizeBytes)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="mt-8">
        <h2 className="mb-2 text-base font-semibold">History</h2>
        {book.history.length === 0 ? (
          <p className="rounded-card border border-dashed border-line-strong px-4 py-6 text-center text-sm text-muted">
            Nothing has happened to this book yet.
          </p>
        ) : (
          <HistoryTable events={book.history} />
        )}
      </section>

      <ReleaseSearchModal
        bookId={book.id}
        bookTitle={book.title}
        open={interactive}
        onClose={() => setInteractive(false)}
      />

      {manualImport && <ManualImportModal book={book} onClose={() => setManualImport(false)} />}

      <ConfirmDialog
        open={confirmDelete}
        title={`Delete “${book.title}”?`}
        danger
        confirmLabel="Delete"
        message="This removes the book from Chaptarr."
        checkbox={{ label: 'Also delete the files from disk' }}
        pending={del.isPending}
        onCancel={() => setConfirmDelete(false)}
        onConfirm={(deleteFiles) =>
          del.mutate(
            { id: book.id, deleteFiles },
            { onSuccess: () => navigate(`/authors/${book.authorId}`) },
          )
        }
      />
    </>
  )
}

export function BookDetailPage() {
  const id = Number(useParams().id)
  const query = useBook(id)
  return <QueryView query={query}>{(book) => <BookBody book={book} />}</QueryView>
}
