import { useQuery } from '@tanstack/react-query'
import { Search } from 'lucide-react'
import { useState } from 'react'
import { Input } from '../components/Form'
import { PageHeader } from '../components/PageHeader'
import { EmptyState, ErrorState, LoadingState } from '../components/States'
import { SegmentedControl } from '../components/Tabs'
import { api } from '../lib/api'
import { useDebouncedValue, useDocumentTitle } from '../lib/hooks'
import { AuthorResultRow } from './add/AuthorResultRow'
import { BookResultRow } from './add/BookResultRow'

type Kind = 'books' | 'authors'

const MIN_CHARS = 2

export function AddPage() {
  useDocumentTitle('Add New')
  const [kind, setKind] = useState<Kind>('books')
  const [text, setText] = useState('')
  const trimmed = text.trim()
  const q = useDebouncedValue(trimmed, 400)
  const ready = q.length >= MIN_CHARS
  // True while the user is still typing, before the debounced query catches up.
  const typing = trimmed !== q

  const common = { staleTime: 5 * 60_000, retry: false, refetchOnWindowFocus: false } as const
  const books = useQuery({
    queryKey: ['search', 'books', q],
    queryFn: () => api.searchBooks(q),
    enabled: ready && kind === 'books',
    ...common,
  })
  const authors = useQuery({
    queryKey: ['search', 'authors', q],
    queryFn: () => api.searchAuthors(q),
    enabled: ready && kind === 'authors',
    ...common,
  })
  const active = kind === 'books' ? books : authors

  let body
  if (!ready) {
    body = (
      <EmptyState
        icon={<Search />}
        title={`Search for ${kind === 'books' ? 'a book' : 'an author'}`}
        description="Results come from the Audible catalogue. Type at least two characters."
      />
    )
  } else if (active.isError) {
    body = <ErrorState title="Search failed" error={active.error} onRetry={() => void active.refetch()} />
  } else if (active.isPending || (typing && !active.data)) {
    body = <LoadingState label="Searching…" />
  } else if (kind === 'books' && books.data) {
    body =
      books.data.length === 0 ? (
        <EmptyState icon={<Search />} title="No books found" description={`Nothing matched “${q}”.`} />
      ) : (
        <ul className="space-y-3">
          {books.data.map((b) => (
            <BookResultRow key={b.asin} book={b} />
          ))}
        </ul>
      )
  } else if (kind === 'authors' && authors.data) {
    body =
      authors.data.length === 0 ? (
        <EmptyState icon={<Search />} title="No authors found" description={`Nothing matched “${q}”.`} />
      ) : (
        <ul className="space-y-3">
          {authors.data.map((a) => (
            <AuthorResultRow key={a.asin} author={a} />
          ))}
        </ul>
      )
  }

  return (
    <>
      <PageHeader title="Add New" description="Find books and authors to add to your library." />
      <div className="mb-6 flex flex-wrap items-center gap-3">
        <SegmentedControl<Kind>
          label="Search for"
          value={kind}
          onChange={setKind}
          options={[
            { value: 'books', label: 'Books' },
            { value: 'authors', label: 'Authors' },
          ]}
        />
        <div className="relative min-w-60 flex-1 sm:max-w-lg">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
          <Input
            autoFocus
            type="search"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder={kind === 'books' ? 'Title, author or series…' : 'Author name…'}
            aria-label="Search"
            className="pl-9"
          />
        </div>
      </div>
      <div className={ready && active.isFetching && active.data ? 'opacity-60 transition-opacity' : undefined}>
        {body}
      </div>
    </>
  )
}
