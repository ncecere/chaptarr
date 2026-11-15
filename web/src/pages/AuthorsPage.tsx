import { EyeOff, LayoutGrid, List, Search, Users } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Cover } from '../components/Cover'
import { Input } from '../components/Form'
import { LinkButton } from '../components/LinkButton'
import { PageHeader } from '../components/PageHeader'
import { ProgressBar } from '../components/ProgressBar'
import { QueryView } from '../components/QueryView'
import { EmptyState, LoadingState } from '../components/States'
import { SegmentedControl } from '../components/Tabs'
import type { Author } from '../lib/api'
import { plural } from '../lib/format'
import { useDocumentTitle, useLocalStorage } from '../lib/hooks'
import { useAuthors } from '../lib/queries'

type View = 'grid' | 'list'

function AuthorProgress({ author }: { author: Author }) {
  const frac = author.bookCount > 0 ? author.downloadedCount / author.bookCount : 0
  return (
    <div className="w-full">
      <ProgressBar value={frac} tone={author.bookCount > 0 && author.downloadedCount === author.bookCount ? 'ok' : 'accent'} />
      <div className="mt-1 flex justify-between text-xs text-muted">
        <span className="tabular-nums">
          {author.downloadedCount} / {author.bookCount} books
        </span>
        {author.missingCount > 0 && <span className="text-warn">{author.missingCount} missing</span>}
      </div>
    </div>
  )
}

function AuthorCard({ author }: { author: Author }) {
  return (
    <Link
      to={`/authors/${author.id}`}
      className="group flex flex-col items-center rounded-card border border-line bg-surface p-4 text-center transition-colors hover:border-accent/50 hover:bg-surface-2"
    >
      <Cover src={author.imageUrl} alt={author.name} kind="author" className="mb-3 size-24" />
      <div className="mb-3 flex w-full items-center justify-center gap-1.5">
        <span className="truncate text-sm font-medium group-hover:text-accent" title={author.name}>
          {author.name}
        </span>
        {!author.monitored && <EyeOff className="size-3.5 shrink-0 text-faint" aria-label="Not monitored" />}
      </div>
      <AuthorProgress author={author} />
    </Link>
  )
}

function AuthorRow({ author }: { author: Author }) {
  return (
    <Link
      to={`/authors/${author.id}`}
      className="group flex items-center gap-4 px-4 py-3 transition-colors hover:bg-surface-2/60"
    >
      <Cover src={author.imageUrl} alt={author.name} kind="author" className="size-11" />
      <div className="flex min-w-0 flex-1 items-center gap-1.5">
        <span className="truncate font-medium group-hover:text-accent">{author.name}</span>
        {!author.monitored && <EyeOff className="size-3.5 shrink-0 text-faint" aria-label="Not monitored" />}
      </div>
      <div className="w-40 shrink-0 sm:w-56">
        <AuthorProgress author={author} />
      </div>
    </Link>
  )
}

export function AuthorsPage() {
  useDocumentTitle('Authors')
  const [view, setView] = useLocalStorage<View>('authors.view', 'grid')
  const [text, setText] = useState('')
  const query = useAuthors()

  const authors = useMemo(() => {
    if (!query.data) return undefined
    const needle = text.trim().toLowerCase()
    return query.data
      .filter((a) => !needle || a.name.toLowerCase().includes(needle))
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [query.data, text])

  return (
    <>
      <PageHeader
        title="Authors"
        description={query.data ? plural(query.data.length, 'author') : undefined}
      />
      <div className="mb-6 flex flex-wrap items-center gap-2">
        <div className="relative min-w-52 flex-1 sm:max-w-xs">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
          <Input
            type="search"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Filter authors"
            aria-label="Filter authors"
            className="pl-9"
          />
        </div>
        <div className="ml-auto">
          <SegmentedControl<View>
            label="View"
            value={view}
            onChange={setView}
            options={[
              { value: 'grid', label: <LayoutGrid className="size-4" />, title: 'Grid' },
              { value: 'list', label: <List className="size-4" />, title: 'List' },
            ]}
          />
        </div>
      </div>
      <QueryView
        query={query}
        data={authors}
        loading={<LoadingState label="Loading authors…" />}
        isEmpty={(a) => a.length === 0}
        empty={
          query.data && query.data.length > 0 ? (
            <EmptyState icon={<Search />} title="No authors match" description="Try a different name." />
          ) : (
            <EmptyState
              icon={<Users />}
              title="No authors yet"
              description="Follow an author and Chaptarr will track their whole catalogue."
              action={
                <LinkButton to="/add">Add New</LinkButton>
              }
            />
          )
        }
      >
        {(list) =>
          view === 'grid' ? (
            <div className="grid grid-cols-[repeat(auto-fill,minmax(11rem,1fr))] gap-4">
              {list.map((a) => (
                <AuthorCard key={a.id} author={a} />
              ))}
            </div>
          ) : (
            <div className="divide-y divide-line overflow-hidden rounded-card border border-line bg-surface">
              {list.map((a) => (
                <AuthorRow key={a.id} author={a} />
              ))}
            </div>
          )
        }
      </QueryView>
    </>
  )
}
