// Typed client for the Chaptarr HTTP API. This file is the contract between
// the Go backend (internal/api) and the UI: every route the server exposes has
// exactly one function here.
//
// Conventions:
//  - All routes live under /api/v1 and speak JSON.
//  - Auth is a same-origin session cookie set by login/setup.
//  - Errors are `{ "error": "message" }` with a non-2xx status.
//  - Timestamps are RFC 3339 strings; sizes are bytes; durations are minutes.

const BASE = '/api/v1'

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(BASE + path, {
    method,
    credentials: 'same-origin',
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (res.status === 204) return undefined as T
  const text = await res.text()
  let data: unknown = undefined
  try {
    data = text ? JSON.parse(text) : undefined
  } catch {
    // Non-JSON body (proxy error page, etc.) falls through to the status text.
  }
  if (!res.ok) {
    const message =
      (data as { error?: string } | undefined)?.error ?? (res.statusText || `HTTP ${res.status}`)
    throw new ApiError(res.status, message)
  }
  return data as T
}

const get = <T>(path: string) => request<T>('GET', path)
const post = <T>(path: string, body?: unknown) => request<T>('POST', path, body ?? {})
const put = <T>(path: string, body: unknown) => request<T>('PUT', path, body)
const del = <T>(path: string) => request<T>('DELETE', path)

function qs(params: Record<string, string | number | boolean | undefined | null>): string {
  const sp = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') sp.set(k, String(v))
  }
  const s = sp.toString()
  return s ? `?${s}` : ''
}

// ---------------------------------------------------------------- auth

export interface User {
  id: number
  username: string
}

export interface AuthStatus {
  /** True until the first admin account has been created. */
  setupRequired: boolean
  authenticated: boolean
  user: User | null
}

export interface Credentials {
  username: string
  password: string
}

// ---------------------------------------------------------------- library

/**
 * missing      monitored, released, no files, nothing in the queue
 * unreleased   release date is in the future
 * downloading  an active queue item exists for the book
 * downloaded   at least one audio file is on disk
 */
export type BookStatus = 'missing' | 'unreleased' | 'downloading' | 'downloaded'

export interface Author {
  id: number
  asin: string
  name: string
  description: string
  imageUrl: string
  monitored: boolean
  /** Automatically add + monitor books released after the author was added. */
  monitorNewBooks: boolean
  addedAt: string
  lastSyncAt: string | null
  bookCount: number
  downloadedCount: number
  missingCount: number
}

export interface SeriesRef {
  id: number
  name: string
  /** Position as the publisher wrote it: "1", "2.5", "1-3". Empty if unknown. */
  position: string
}

export interface Book {
  id: number
  asin: string
  title: string
  subtitle: string
  authorId: number
  authorName: string
  /** All credited authors; the first is the library author. */
  authors: string[]
  narrators: string[]
  series: SeriesRef | null
  description: string
  coverUrl: string
  /** YYYY-MM-DD, or "" when unknown. */
  releaseDate: string
  runtimeMinutes: number
  language: string
  publisher: string
  monitored: boolean
  status: BookStatus
  /** Directory holding the book's files, "" when not downloaded. */
  path: string
  sizeBytes: number
  /** Primary container of the files on disk: "m4b", "mp3", ... or "". */
  format: string
  addedAt: string
}

export interface BookFile {
  id: number
  path: string
  sizeBytes: number
  format: string
}

export interface BookDetail extends Book {
  files: BookFile[]
  history: HistoryEvent[]
}

export interface AuthorDetail extends Author {
  books: Book[]
}

export interface Series {
  id: number
  asin: string
  name: string
  bookCount: number
  downloadedCount: number
  /** Cover of the first book, for the series grid. */
  coverUrl: string
  authorName: string
}

export interface SeriesDetail extends Series {
  books: Book[]
}

export interface BookFilter {
  status?: BookStatus
  authorId?: number
  monitored?: boolean
  /** Case-insensitive match on title, author, narrator, or series. */
  q?: string
}

// ---------------------------------------------------------------- metadata search

/** A book from the metadata provider that may or may not be in the library. */
export interface BookResult {
  asin: string
  title: string
  subtitle: string
  authors: { asin: string; name: string }[]
  narrators: string[]
  seriesName: string
  seriesPosition: string
  description: string
  coverUrl: string
  releaseDate: string
  runtimeMinutes: number
  language: string
  publisher: string
  /** Library book id when already added, otherwise null. */
  libraryBookId: number | null
}

export interface AuthorResult {
  asin: string
  name: string
  imageUrl: string
  description: string
  /** Library author id when already added, otherwise null. */
  libraryAuthorId: number | null
}

/** all: monitor every book. future: only books not yet released. none: add unmonitored. */
export type MonitorMode = 'all' | 'future' | 'none'

export interface AddAuthorRequest {
  asin: string
  /** Sent so the server can add the author even if the bio lookup fails. */
  name: string
  monitor: MonitorMode
  monitorNewBooks: boolean
  /** Start an indexer search for every monitored missing book right away. */
  searchOnAdd: boolean
}

export interface AddBookRequest {
  asin: string
  monitored: boolean
  searchOnAdd: boolean
}

// ---------------------------------------------------------------- releases / queue / history

export type Protocol = 'torrent' | 'usenet'

/** One search result from an indexer, already scored against a book. */
export interface Release {
  /** Opaque id; pass back to grabRelease. Valid for ~30 minutes after the search. */
  id: string
  title: string
  indexer: string
  protocol: Protocol
  sizeBytes: number
  /** Null for usenet. */
  seeders: number | null
  leechers: number | null
  /** Age of the post in days. */
  ageDays: number
  /** Detected audio format ("m4b", "mp3", "flac", ...) or "" if unknown. */
  format: string
  /** Higher is better; the list arrives sorted by score descending. */
  score: number
  /** Non-empty when automatic search would skip this release. Manual grab still works. */
  rejections: string[]
  infoUrl: string
}

export type QueueStatus =
  | 'queued'
  | 'downloading'
  | 'paused'
  | 'completed' // finished in the client, waiting for import
  | 'importing'
  | 'failed'

export interface QueueItem {
  id: number
  bookId: number
  bookTitle: string
  authorName: string
  coverUrl: string
  releaseTitle: string
  indexer: string
  protocol: Protocol
  downloadClient: string
  status: QueueStatus
  /** 0..1 */
  progress: number
  sizeBytes: number
  /** Seconds remaining, null when unknown. */
  etaSeconds: number | null
  /** Set when status is "failed", or when an import attempt failed and will retry. */
  error: string
  addedAt: string
}

export type HistoryEventType =
  | 'grabbed'
  | 'imported'
  | 'downloadFailed'
  | 'importFailed'
  | 'fileDeleted'

export interface HistoryEvent {
  id: number
  bookId: number
  bookTitle: string
  authorName: string
  type: HistoryEventType
  /** Release title for grab/import events, file path for deletes. */
  sourceTitle: string
  /** Human-readable detail: indexer + client for grabs, error text for failures. */
  message: string
  date: string
}

export interface Page<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

// ---------------------------------------------------------------- settings

export interface RootFolder {
  id: number
  path: string
  /** False when the directory is missing or not writable. */
  accessible: boolean
  freeBytes: number
  totalBytes: number
}

export type IndexerType = 'torznab' | 'newznab'

export interface Indexer {
  id: number
  name: string
  type: IndexerType
  /** Base URL of the feed, e.g. http://prowlarr:9696/1/api */
  url: string
  /** Write-only: the server returns "" when a key is stored; send "" to keep it. */
  apiKey: string
  hasApiKey: boolean
  /** Comma-separated newznab category ids. Default "3030". */
  categories: string
  enabled: boolean
  /** 1 (highest) .. 50. Breaks score ties between indexers. */
  priority: number
}

export type IndexerInput = Omit<Indexer, 'id' | 'hasApiKey'>

export type DownloadClientType = 'qbittorrent' | 'sabnzbd'

export interface DownloadClient {
  id: number
  name: string
  type: DownloadClientType
  /** Base URL, e.g. http://localhost:8080 */
  url: string
  /** qBittorrent only. */
  username: string
  /** qBittorrent only. Write-only, same rule as Indexer.apiKey. */
  password: string
  hasPassword: boolean
  /** SABnzbd only. Write-only, same rule as Indexer.apiKey. */
  apiKey: string
  hasApiKey: boolean
  /** Category/label applied to downloads. Default "audiobooks". */
  category: string
  enabled: boolean
}

export type DownloadClientInput = Omit<DownloadClient, 'id' | 'hasPassword' | 'hasApiKey'>

export interface TestResult {
  ok: boolean
  message: string
}

export interface Settings {
  /** Audible marketplace used for metadata: us, uk, ca, au, de, fr, es, it, in, jp. */
  audibleRegion: string
  /** Only releases/books in this language are auto-added. "" means any. */
  language: string
  /**
   * Folder template relative to the root folder. Tokens: {Author}, {Series},
   * {SeriesPosition}, {Title}, {Year}, {Narrator}, {ASIN}. Path segments that
   * end up empty (e.g. {Series} for a standalone book) are dropped.
   */
  folderFormat: string
  /** Audio formats, most preferred first, e.g. ["m4b","mp3","flac"]. */
  preferredFormats: string[]
  /** hardlink falls back to copy when source and library are on different filesystems. */
  importMode: 'hardlink' | 'copy' | 'move'
  /** Torrents with fewer seeders are rejected by automatic search. */
  minSeeders: number
  /** Minutes between automatic searches for missing books. 0 disables. */
  missingSearchIntervalMinutes: number
  /** Hours between metadata refreshes for monitored authors. 0 disables. */
  authorRefreshIntervalHours: number
  /** Id of the root folder new books import into. Null until one exists. */
  defaultRootFolderId: number | null
}

// ---------------------------------------------------------------- system

export interface SystemStatus {
  version: string
  /** Directory holding the database and config. */
  dataDir: string
  startedAt: string
  authorCount: number
  bookCount: number
  downloadedCount: number
  missingCount: number
  queueCount: number
  librarySizeBytes: number
  /** Setup checklist flags for the first-run banner. */
  hasRootFolder: boolean
  hasIndexer: boolean
  hasDownloadClient: boolean
  /** The current user's API key for X-Api-Key access. */
  apiKey: string
}

export type TaskName =
  | 'refreshAuthors'
  | 'searchMissing'
  | 'syncQueue'
  | 'scanLibrary'

export interface Task {
  name: TaskName
  label: string
  /** 0 when the task only runs on demand. */
  intervalSeconds: number
  running: boolean
  lastRunAt: string | null
  lastDurationMs: number
  /** Empty on success. */
  lastError: string
  nextRunAt: string | null
}

export interface LogEntry {
  time: string
  level: 'debug' | 'info' | 'warn' | 'error'
  message: string
}

export interface DirectoryListing {
  path: string
  /** Null at the filesystem root. */
  parent: string | null
  directories: { name: string; path: string }[]
}

// ---------------------------------------------------------------- routes

export const api = {
  // auth
  authStatus: () => get<AuthStatus>('/auth/status'),
  /** Creates the first admin account and signs in. 409 once setup is complete. */
  setup: (c: Credentials) => post<AuthStatus>('/auth/setup', c),
  login: (c: Credentials) => post<AuthStatus>('/auth/login', c),
  logout: () => post<void>('/auth/logout'),
  changePassword: (currentPassword: string, newPassword: string) =>
    post<void>('/auth/password', { currentPassword, newPassword }),

  // metadata search (not yet in library)
  searchBooks: (q: string) => get<BookResult[]>(`/search/books${qs({ q })}`),
  searchAuthors: (q: string) => get<AuthorResult[]>(`/search/authors${qs({ q })}`),

  // authors
  authors: () => get<Author[]>('/authors'),
  author: (id: number) => get<AuthorDetail>(`/authors/${id}`),
  /** Adds the author and imports their catalogue. Returns once the books are in. */
  addAuthor: (r: AddAuthorRequest) => post<AuthorDetail>('/authors', r),
  updateAuthor: (id: number, patch: { monitored?: boolean; monitorNewBooks?: boolean }) =>
    put<Author>(`/authors/${id}`, patch),
  deleteAuthor: (id: number, deleteFiles: boolean) =>
    del<void>(`/authors/${id}${qs({ deleteFiles })}`),
  /** Re-pulls the author's catalogue from the metadata provider. */
  refreshAuthor: (id: number) => post<AuthorDetail>(`/authors/${id}/refresh`),
  /** Queues an indexer search for every monitored missing book by the author. */
  searchAuthor: (id: number) => post<{ queued: number }>(`/authors/${id}/search`),

  // books
  books: (f: BookFilter = {}) => get<Book[]>(`/books${qs({ ...f })}`),
  book: (id: number) => get<BookDetail>(`/books/${id}`),
  /** Adds a single book (and its author, unmonitored, if new). */
  addBook: (r: AddBookRequest) => post<Book>('/books', r),
  updateBook: (id: number, patch: { monitored?: boolean }) => put<Book>(`/books/${id}`, patch),
  /** Bulk monitor toggle, used by the author page's select-all. */
  setBooksMonitored: (ids: number[], monitored: boolean) =>
    put<void>('/books/monitor', { ids, monitored }),
  deleteBook: (id: number, deleteFiles: boolean) =>
    del<void>(`/books/${id}${qs({ deleteFiles })}`),
  /** Interactive search: queries every enabled indexer. Can take 10+ seconds. */
  bookReleases: (id: number) => get<Release[]>(`/books/${id}/releases`),
  /** Sends a release from bookReleases to a download client. */
  grabRelease: (bookId: number, releaseId: string) =>
    post<QueueItem>(`/books/${bookId}/grab`, { releaseId }),
  /** Automatic search: picks the best acceptable release and grabs it. */
  searchBook: (id: number) => post<{ grabbed: boolean; message: string }>(`/books/${id}/search`),

  // series
  series: () => get<Series[]>('/series'),
  seriesDetail: (id: number) => get<SeriesDetail>(`/series/${id}`),

  // activity
  queue: () => get<QueueItem[]>('/queue'),
  /** removeFromClient also deletes the download (and its data) in the client. */
  removeQueueItem: (id: number, removeFromClient: boolean) =>
    del<void>(`/queue/${id}${qs({ removeFromClient })}`),
  /** Retries the import of a completed download right now. */
  retryImport: (id: number) => post<void>(`/queue/${id}/import`),
  history: (page = 1, pageSize = 50) =>
    get<Page<HistoryEvent>>(`/history${qs({ page, pageSize })}`),

  // settings: root folders
  rootFolders: () => get<RootFolder[]>('/rootfolders'),
  addRootFolder: (path: string) => post<RootFolder>('/rootfolders', { path }),
  deleteRootFolder: (id: number) => del<void>(`/rootfolders/${id}`),
  /** Server-side folder browser for the root folder picker. Omit path for a sensible start. */
  browse: (path?: string) => get<DirectoryListing>(`/filesystem${qs({ path })}`),

  // settings: indexers
  indexers: () => get<Indexer[]>('/indexers'),
  addIndexer: (i: IndexerInput) => post<Indexer>('/indexers', i),
  updateIndexer: (id: number, i: IndexerInput) => put<Indexer>(`/indexers/${id}`, i),
  deleteIndexer: (id: number) => del<void>(`/indexers/${id}`),
  /** Tests unsaved form values. Pass id so a blank apiKey falls back to the stored one. */
  testIndexer: (i: IndexerInput, id?: number) =>
    post<TestResult>('/indexers/test', { ...i, id: id ?? null }),

  // settings: download clients
  downloadClients: () => get<DownloadClient[]>('/downloadclients'),
  addDownloadClient: (c: DownloadClientInput) => post<DownloadClient>('/downloadclients', c),
  updateDownloadClient: (id: number, c: DownloadClientInput) =>
    put<DownloadClient>(`/downloadclients/${id}`, c),
  deleteDownloadClient: (id: number) => del<void>(`/downloadclients/${id}`),
  testDownloadClient: (c: DownloadClientInput, id?: number) =>
    post<TestResult>('/downloadclients/test', { ...c, id: id ?? null }),

  // settings: general
  settings: () => get<Settings>('/settings'),
  updateSettings: (s: Settings) => put<Settings>('/settings', s),

  // system
  status: () => get<SystemStatus>('/system/status'),
  tasks: () => get<Task[]>('/system/tasks'),
  /** Starts the task in the background; poll tasks() for completion. */
  runTask: (name: TaskName) => post<void>(`/system/tasks/${name}/run`),
  /** Most recent entries first. */
  logs: (limit = 200) => get<LogEntry[]>(`/system/logs${qs({ limit })}`),
  regenerateApiKey: () => post<{ apiKey: string }>('/system/apikey'),
}
