import { History } from 'lucide-react'
import { useState } from 'react'
import { HistoryTable } from '../../components/HistoryTable'
import { Pagination } from '../../components/Pagination'
import { QueryView } from '../../components/QueryView'
import { EmptyState } from '../../components/States'
import { useHistory } from '../../lib/queries'

const PAGE_SIZE = 50

export function HistoryTab() {
  const [page, setPage] = useState(1)
  const query = useHistory(page, PAGE_SIZE)
  return (
    <QueryView
      query={query}
      isEmpty={(p) => p.total === 0}
      empty={
        <EmptyState
          icon={<History />}
          title="No history yet"
          description="Grabs, imports and failures will be listed here."
        />
      }
    >
      {(data) => (
        <>
          <HistoryTable events={data.items} showBook />
          <Pagination
            page={data.page}
            pageSize={data.pageSize || PAGE_SIZE}
            total={data.total}
            onPage={setPage}
            busy={query.isFetching}
          />
        </>
      )}
    </QueryView>
  )
}
