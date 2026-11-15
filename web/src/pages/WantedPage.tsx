import { CircleCheck, Search } from 'lucide-react'
import { BookTable } from '../components/BookTable'
import { Button } from '../components/Button'
import { PageHeader } from '../components/PageHeader'
import { QueryView } from '../components/QueryView'
import { EmptyState, LoadingState } from '../components/States'
import { plural } from '../lib/format'
import { useDocumentTitle } from '../lib/hooks'
import { useRunTask } from '../lib/mutations'
import { useBooks } from '../lib/queries'

export function WantedPage() {
  useDocumentTitle('Wanted')
  const query = useBooks({ status: 'missing', monitored: true })
  const run = useRunTask()
  const count = query.data?.length ?? 0

  return (
    <>
      <PageHeader
        title="Wanted"
        description={query.data ? `${plural(count, 'missing book')}` : undefined}
        actions={
          <Button
            variant="primary"
            icon={<Search className="size-4" />}
            loading={run.isPending}
            disabled={count === 0}
            onClick={() => run.mutate({ name: 'searchMissing', label: 'search for all missing books' })}
          >
            Search All
          </Button>
        }
      />
      <QueryView
        query={query}
        loading={<LoadingState label="Loading wanted books…" />}
        isEmpty={(b) => b.length === 0}
        empty={
          <EmptyState
            icon={<CircleCheck />}
            title="Nothing missing"
            description="Every monitored, released book is downloaded or on its way."
          />
        }
      >
        {(books) => <BookTable books={books} showCover showAuthor showSeries showMonitor showSearch />}
      </QueryView>
    </>
  )
}
