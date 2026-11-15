import type { UseQueryResult } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { ErrorState, LoadingState } from './States'

interface QueryViewProps<T> {
  query: UseQueryResult<T>
  children: (data: T) => ReactNode
  /** Derived data (filtered/sorted) to render in place of query.data. */
  data?: T
  loading?: ReactNode
  /** Rendered instead of children when isEmpty(data) is true. */
  empty?: ReactNode
  isEmpty?: (data: T) => boolean
}

/** Standard loading / error / empty handling for a query. */
export function QueryView<T>({ query, children, data: derived, loading, empty, isEmpty }: QueryViewProps<T>) {
  const data = derived ?? query.data
  if (data === undefined) {
    if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />
    return <>{loading ?? <LoadingState />}</>
  }
  if (isEmpty?.(data)) return <>{empty}</>
  return <>{children(data)}</>
}
