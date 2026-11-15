import { BookOpen, LayoutGrid, List, Search, ArrowDownAZ, ArrowUpAZ } from 'lucide-react'
import { useMemo, useState } from 'react'
import { BookCard } from '../components/BookCard'
import { BookTable } from '../components/BookTable'
import { IconButton } from '../components/Button'
import { CoverGrid, CoverGridSkeleton } from '../components/CoverGrid'
import { Input, Select } from '../components/Form'
import { LinkButton } from '../components/LinkButton'
import { PageHeader } from '../components/PageHeader'
import { QueryView } from '../components/QueryView'
import { EmptyState } from '../components/States'
import { SegmentedControl } from '../components/Tabs'
import type { Book, BookStatus } from '../lib/api'
import { useDebouncedValue, useDocumentTitle, useLocalStorage } from '../lib/hooks'
import { plural } from '../lib/format'
import { useBooks } from '../lib/queries'

type SortKey = 'title' | 'author' | 'releaseDate' | 'addedAt'
type View = 'grid' | 'table'

const SORTS: { value: SortKey; label: string }[] = [
  { value: 'title', label: 'Title' },
  { value: 'author', label: 'Author' },
  { value: 'releaseDate', label: 'Release date' },
  { value: 'addedAt', label: 'Date added' },
]

const STATUSES: { value: '' | BookStatus; label: string }[] = [
  { value: '', label: 'All statuses' },
  { value: 'downloaded', label: 'Downloaded' },
  { value: 'downloading', label: 'Downloading' },
  { value: 'missing', label: 'Missing' },
  { value: 'unreleased', label: 'Unreleased' },
]

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

function sortBooks(books: Book[], key: SortKey, dir: 'asc' | 'desc'): Book[] {
  const sign = dir === 'asc' ? 1 : -1
  const cmp = (a: Book, b: Book): number => {
    switch (key) {
      case 'title':
        return collator.compare(a.title, b.title)
      case 'author':
        return collator.compare(a.authorName, b.authorName) || collator.compare(a.title, b.title)
      case 'releaseDate':
        // Books without a date always sort last, whatever the direction.
        if (!a.releaseDate || !b.releaseDate) {
          return a.releaseDate === b.releaseDate ? 0 : a.releaseDate ? -sign : sign
        }
        return a.releaseDate < b.releaseDate ? -1 : a.releaseDate > b.releaseDate ? 1 : 0
      case 'addedAt':
        return a.addedAt < b.addedAt ? -1 : a.addedAt > b.addedAt ? 1 : 0
    }
  }
  return [...books].sort((a, b) => sign * cmp(a, b))
}

export function LibraryPage() {
  useDocumentTitle('Library')
  const [view, setView] = useLocalStorage<View>('library.view', 'grid')
  const [sort, setSort] = useLocalStorage<SortKey>('library.sort', 'title')
  const [dir, setDir] = useLocalStorage<'asc' | 'desc'>('library.dir', 'asc')
  const [text, setText] = useState('')
  const [status, setStatus] = useState<'' | BookStatus>('')
  const q = useDebouncedValue(text.trim(), 250)

  const query = useBooks({ q: q || undefined, status: status || undefined })
  const filtered = q !== '' || status !== ''
  const books = useMemo(
    () => (query.data ? sortBooks(query.data, sort, dir) : undefined),
    [query.data, sort, dir],
  )

  return (
    <>
      <PageHeader
        title="Library"
        description={query.data ? plural(query.data.length, 'book') + (filtered ? ' match' : '') : undefined}
      />

      <div className="mb-6 flex flex-wrap items-center gap-2">
        <div className="relative min-w-52 flex-1 sm:max-w-xs">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
          <Input
            type="search"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Filter by title, author, narrator, series"
            aria-label="Filter books"
            className="pl-9"
          />
        </div>
        <Select
          value={status}
          onChange={(e) => setStatus(e.target.value as '' | BookStatus)}
          aria-label="Filter by status"
          className="w-auto"
        >
          {STATUSES.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </Select>
        <div className="flex items-center gap-1">
          <Select
            value={sort}
            onChange={(e) => setSort(e.target.value as SortKey)}
            aria-label="Sort by"
            className="w-auto"
          >
            {SORTS.map((s) => (
              <option key={s.value} value={s.value}>
                Sort: {s.label}
              </option>
            ))}
          </Select>
          <IconButton
            label={dir === 'asc' ? 'Ascending — switch to descending' : 'Descending — switch to ascending'}
            variant="secondary"
            onClick={() => setDir(dir === 'asc' ? 'desc' : 'asc')}
          >
            {dir === 'asc' ? <ArrowDownAZ className="size-4" /> : <ArrowUpAZ className="size-4" />}
          </IconButton>
        </div>
        <div className="ml-auto">
          <SegmentedControl<View>
            label="View"
            value={view}
            onChange={setView}
            options={[
              { value: 'grid', label: <LayoutGrid className="size-4" />, title: 'Cover grid' },
              { value: 'table', label: <List className="size-4" />, title: 'Table' },
            ]}
          />
        </div>
      </div>

      <QueryView
        query={query}
        data={books}
        loading={<CoverGridSkeleton />}
        isEmpty={(b) => b.length === 0}
        empty={
          filtered ? (
            <EmptyState
              icon={<Search />}
              title="No books match"
              description="Try a different search term or status."
            />
          ) : (
            <EmptyState
              icon={<BookOpen />}
              title="Your library is empty"
              description="Search for an author or a book to start building your audiobook library."
              action={
                <LinkButton to="/add">Add New</LinkButton>
              }
            />
          )
        }
      >
        {(list) =>
          view === 'grid' ? (
            <CoverGrid>
              {list.map((b) => (
                <BookCard key={b.id} book={b} />
              ))}
            </CoverGrid>
          ) : (
            <BookTable books={list} showCover showAuthor showSeries showSize showMonitor />
          )
        }
      </QueryView>
    </>
  )
}
