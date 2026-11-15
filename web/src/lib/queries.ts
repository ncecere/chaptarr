import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { api, type BookFilter } from './api'

// Query hooks, one per read endpoint. Keys are the first array element so
// mutations can invalidate whole families (see mutations.ts).

export function useStatus() {
  return useQuery({ queryKey: ['status'], queryFn: api.status, refetchInterval: 10_000 })
}

export function useAuthors() {
  return useQuery({ queryKey: ['authors'], queryFn: api.authors })
}

export function useAuthor(id: number) {
  return useQuery({ queryKey: ['author', id], queryFn: () => api.author(id) })
}

export function useBooks(filter: BookFilter = {}) {
  return useQuery({
    queryKey: ['books', filter],
    queryFn: () => api.books(filter),
    placeholderData: keepPreviousData,
  })
}

export function useBook(id: number) {
  return useQuery({ queryKey: ['book', id], queryFn: () => api.book(id) })
}

export function useSeriesList() {
  return useQuery({ queryKey: ['series'], queryFn: api.series })
}

export function useSeriesDetail(id: number) {
  return useQuery({ queryKey: ['seriesDetail', id], queryFn: () => api.seriesDetail(id) })
}

/** Polls every 5s; TanStack pauses interval refetching while the tab is hidden. */
export function useQueue() {
  return useQuery({ queryKey: ['queue'], queryFn: api.queue, refetchInterval: 5_000 })
}

export function useHistory(page: number, pageSize = 50) {
  return useQuery({
    queryKey: ['history', page, pageSize],
    queryFn: () => api.history(page, pageSize),
    placeholderData: keepPreviousData,
  })
}

export function useRootFolders() {
  return useQuery({ queryKey: ['rootFolders'], queryFn: api.rootFolders })
}

export function useIndexers() {
  return useQuery({ queryKey: ['indexers'], queryFn: api.indexers })
}

export function useDownloadClients() {
  return useQuery({ queryKey: ['downloadClients'], queryFn: api.downloadClients })
}

export function useSettings() {
  return useQuery({ queryKey: ['settings'], queryFn: api.settings })
}

export function useBrowse(path: string | undefined) {
  return useQuery({
    queryKey: ['browse', path ?? ''],
    queryFn: () => api.browse(path),
    staleTime: 0,
    placeholderData: keepPreviousData,
  })
}

export function useLogs(limit: number, autoRefresh: boolean) {
  return useQuery({
    queryKey: ['logs', limit],
    queryFn: () => api.logs(limit),
    refetchInterval: autoRefresh ? 5_000 : false,
  })
}
