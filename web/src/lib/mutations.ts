import { useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query'
import {
  api,
  type AddAuthorRequest,
  type AddSeriesBooksRequest,
  type AddBookRequest,
  type DownloadClientInput,
  type IndexerInput,
  type ManualImportRequest,
  type Settings,
  type SystemStatus,
  type TaskName,
} from './api'
import { plural } from './format'
import { toast } from './toast'

// Mutation hooks. Each invalidates the query families it can affect so screens
// update without a reload, and reports the outcome as a toast. Failures are
// toasted centrally by the query client's MutationCache.

const LIBRARY_KEYS = [
  'authors',
  'author',
  'books',
  'book',
  'series',
  'seriesDetail',
  'status',
  'queue',
  'history',
]

function invalidateLibrary(qc: QueryClient) {
  return Promise.all(LIBRARY_KEYS.map((k) => qc.invalidateQueries({ queryKey: [k] })))
}

// ---------------------------------------------------------------- books

export function useUpdateBook() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: number; monitored: boolean }) =>
      api.updateBook(v.id, { monitored: v.monitored }),
    onSuccess: () => invalidateLibrary(qc),
  })
}

export function useSetBooksMonitored() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { ids: number[]; monitored: boolean }) =>
      api.setBooksMonitored(v.ids, v.monitored),
    onSuccess: (_d, v) => {
      toast.success(`${plural(v.ids.length, 'book')} ${v.monitored ? 'monitored' : 'unmonitored'}`)
      return invalidateLibrary(qc)
    },
  })
}

export function useDeleteBook() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: number; deleteFiles: boolean }) => api.deleteBook(v.id, v.deleteFiles),
    onSuccess: () => {
      toast.success('Book deleted')
      // Not awaited: the caller navigates away before the deleted book's detail query refetches.
      void invalidateLibrary(qc)
    },
  })
}

export function useSearchBook() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.searchBook(id),
    onSuccess: (r) => {
      if (r.grabbed) toast.success(r.message || 'Release grabbed')
      else toast.info(r.message || 'No suitable release found')
      return invalidateLibrary(qc)
    },
  })
}

export function useGrabRelease() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { bookId: number; releaseId: string }) => api.grabRelease(v.bookId, v.releaseId),
    onSuccess: (item) => {
      toast.success(`Sent to ${item.downloadClient || 'download client'}`)
      return invalidateLibrary(qc)
    },
  })
}

export function useAddBook() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (r: AddBookRequest) => api.addBook(r),
    onSuccess: (book) => {
      toast.success(`Added “${book.title}”`)
      return invalidateLibrary(qc)
    },
  })
}

// ---------------------------------------------------------------- authors

export function useAddAuthor() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (r: AddAuthorRequest) => api.addAuthor(r),
    onSuccess: (author) => {
      toast.success(`Added ${author.name} with ${plural(author.books.length, 'book')}`)
      return invalidateLibrary(qc)
    },
  })
}

/** Imports the full catalogue of an author who is already in the library. */
export function useImportAuthorCatalogue() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { before: number; request: AddAuthorRequest }) => api.addAuthor(v.request),
    onSuccess: (author, v) => {
      const added = author.books.length - v.before
      if (added > 0) toast.success(`Added ${plural(added, 'book')} by ${author.name}`)
      else toast.info(`No other books by ${author.name} were found`)
      return invalidateLibrary(qc)
    },
  })
}

export function useAddSeriesBooks() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: number; request: AddSeriesBooksRequest }) => api.addSeriesBooks(v.id, v.request),
    onSuccess: (r, v) => {
      toast.success(`Added ${plural(r.added, 'book')} to the series`)
      void qc.invalidateQueries({ queryKey: ['seriesMissing', v.id] })
      return invalidateLibrary(qc)
    },
  })
}

export function useUpdateAuthor() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: number; patch: { monitored?: boolean; monitorNewBooks?: boolean } }) =>
      api.updateAuthor(v.id, v.patch),
    onSuccess: () => invalidateLibrary(qc),
  })
}

export function useDeleteAuthor() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: number; deleteFiles: boolean }) => api.deleteAuthor(v.id, v.deleteFiles),
    onSuccess: () => {
      toast.success('Author deleted')
      void invalidateLibrary(qc)
    },
  })
}

export function useRefreshAuthor() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.refreshAuthor(id),
    onSuccess: (author) => {
      qc.setQueryData(['author', author.id], author)
      toast.success(`Refreshed ${author.name}`)
      return invalidateLibrary(qc)
    },
  })
}

export function useSearchAuthor() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.searchAuthor(id),
    onSuccess: (r) => {
      if (r.queued > 0) toast.success(`Searching for ${plural(r.queued, 'missing book')}`)
      else toast.info('No monitored missing books to search for')
      return qc.invalidateQueries({ queryKey: ['queue'] })
    },
  })
}

// ---------------------------------------------------------------- activity

export function useRemoveQueueItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: number; removeFromClient: boolean }) =>
      api.removeQueueItem(v.id, v.removeFromClient),
    onSuccess: () => {
      toast.success('Removed from queue')
      return invalidateLibrary(qc)
    },
  })
}

export function useRetryImport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.retryImport(id),
    onSuccess: () => {
      toast.success('Import retried')
      return invalidateLibrary(qc)
    },
  })
}

export function useManualImport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { bookId: number; request: ManualImportRequest }) => api.manualImport(v.bookId, v.request),
    meta: { silent: true },
    onSuccess: (book) => {
      toast.success(`Imported “${book.title}”`)
      return invalidateLibrary(qc)
    },
  })
}

// ---------------------------------------------------------------- settings

export function useUpdateSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (s: Settings) => api.updateSettings(s),
    onSuccess: (s) => {
      qc.setQueryData(['settings'], s)
      toast.success('Settings saved')
    },
  })
}

export function useTestMetadataSource() {
  return useMutation({ mutationFn: api.testMetadataSource, meta: { silent: true } })
}

export function useAddRootFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (path: string) => api.addRootFolder(path),
    onSuccess: (f) => {
      toast.success(`Added root folder ${f.path}`)
      return Promise.all([
        qc.invalidateQueries({ queryKey: ['rootFolders'] }),
        qc.invalidateQueries({ queryKey: ['settings'] }),
        qc.invalidateQueries({ queryKey: ['status'] }),
      ])
    },
  })
}

export function useDeleteRootFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.deleteRootFolder(id),
    onSuccess: () => {
      toast.success('Root folder removed')
      return Promise.all([
        qc.invalidateQueries({ queryKey: ['rootFolders'] }),
        qc.invalidateQueries({ queryKey: ['settings'] }),
        qc.invalidateQueries({ queryKey: ['status'] }),
      ])
    },
  })
}

export function useSaveIndexer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id?: number; input: IndexerInput; silentSuccess?: boolean }) =>
      v.id === undefined ? api.addIndexer(v.input) : api.updateIndexer(v.id, v.input),
    onSuccess: (_i, v) => {
      if (!v.silentSuccess) toast.success(v.id === undefined ? 'Indexer added' : 'Indexer saved')
      return Promise.all([
        qc.invalidateQueries({ queryKey: ['indexers'] }),
        qc.invalidateQueries({ queryKey: ['status'] }),
      ])
    },
  })
}

export function useDeleteIndexer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.deleteIndexer(id),
    onSuccess: () => {
      toast.success('Indexer deleted')
      return Promise.all([
        qc.invalidateQueries({ queryKey: ['indexers'] }),
        qc.invalidateQueries({ queryKey: ['status'] }),
      ])
    },
  })
}

export function useTestIndexer() {
  return useMutation({
    mutationFn: (v: { input: IndexerInput; id?: number }) => api.testIndexer(v.input, v.id),
    meta: { silent: true },
  })
}

export function useSaveDownloadClient() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id?: number; input: DownloadClientInput; silentSuccess?: boolean }) =>
      v.id === undefined ? api.addDownloadClient(v.input) : api.updateDownloadClient(v.id, v.input),
    onSuccess: (_c, v) => {
      if (!v.silentSuccess)
        toast.success(v.id === undefined ? 'Download client added' : 'Download client saved')
      return Promise.all([
        qc.invalidateQueries({ queryKey: ['downloadClients'] }),
        qc.invalidateQueries({ queryKey: ['status'] }),
      ])
    },
  })
}

export function useDeleteDownloadClient() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.deleteDownloadClient(id),
    onSuccess: () => {
      toast.success('Download client deleted')
      return Promise.all([
        qc.invalidateQueries({ queryKey: ['downloadClients'] }),
        qc.invalidateQueries({ queryKey: ['status'] }),
      ])
    },
  })
}

export function useTestDownloadClient() {
  return useMutation({
    mutationFn: (v: { input: DownloadClientInput; id?: number }) =>
      api.testDownloadClient(v.input, v.id),
    meta: { silent: true },
  })
}

// ---------------------------------------------------------------- system

export function useRunTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { name: TaskName; label?: string }) => api.runTask(v.name),
    onSuccess: (_d, v) => {
      toast.success(`Started ${v.label ?? v.name}`)
      return qc.invalidateQueries({ queryKey: ['tasks'] })
    },
  })
}

export function useRegenerateApiKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.regenerateApiKey(),
    onSuccess: ({ apiKey }) => {
      qc.setQueryData<SystemStatus>(['status'], (s) => (s ? { ...s, apiKey } : s))
      toast.success('API key regenerated')
    },
  })
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (v: { currentPassword: string; newPassword: string }) =>
      api.changePassword(v.currentPassword, v.newPassword),
    onSuccess: () => toast.success('Password changed'),
  })
}
