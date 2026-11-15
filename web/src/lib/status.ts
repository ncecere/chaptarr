import type { BookStatus, HistoryEventType, MetadataSourceType, QueueStatus } from './api'

export type Tone = 'neutral' | 'accent' | 'ok' | 'warn' | 'bad' | 'info'

export const bookStatusMeta: Record<BookStatus, { label: string; tone: Tone }> = {
  missing: { label: 'Missing', tone: 'bad' },
  unreleased: { label: 'Unreleased', tone: 'info' },
  downloading: { label: 'Downloading', tone: 'accent' },
  downloaded: { label: 'Downloaded', tone: 'ok' },
}

export const queueStatusMeta: Record<QueueStatus, { label: string; tone: Tone }> = {
  queued: { label: 'Queued', tone: 'neutral' },
  downloading: { label: 'Downloading', tone: 'accent' },
  paused: { label: 'Paused', tone: 'warn' },
  completed: { label: 'Awaiting import', tone: 'info' },
  importing: { label: 'Importing', tone: 'info' },
  failed: { label: 'Failed', tone: 'bad' },
}

export const historyTypeMeta: Record<HistoryEventType, { label: string; tone: Tone }> = {
  grabbed: { label: 'Grabbed', tone: 'accent' },
  imported: { label: 'Imported', tone: 'ok' },
  downloadFailed: { label: 'Download failed', tone: 'bad' },
  importFailed: { label: 'Import failed', tone: 'bad' },
  fileDeleted: { label: 'File deleted', tone: 'neutral' },
}

/** Numeric value of a series position as the publisher wrote it ("2.5", "1-3"); Infinity if unknown. */
export function seriesPositionValue(position: string | undefined): number {
  const m = position ? /\d+(\.\d+)?/.exec(position) : null
  return m ? parseFloat(m[0]) : Number.POSITIVE_INFINITY
}

/** Display names and catalogue-id prefixes of the metadata sources. */
export const metadataSourceMeta: Record<MetadataSourceType, { label: string; idLabel: string; prefix: string }> = {
  audible: { label: 'Audible', idLabel: 'ASIN', prefix: '' },
  itunes: { label: 'iTunes', idLabel: 'iTunes ID', prefix: 'it:' },
  openlibrary: { label: 'Open Library', idLabel: 'Open Library ID', prefix: 'ol:' },
  googlebooks: { label: 'Google Books', idLabel: 'Google Books ID', prefix: 'gb:' },
}

/** The source a catalogue id belongs to; ids from sources other than Audible are prefixed. */
export function sourceOfId(id: string): MetadataSourceType {
  for (const type of ['itunes', 'openlibrary', 'googlebooks'] as const) {
    if (id.startsWith(metadataSourceMeta[type].prefix)) return type
  }
  return 'audible'
}
