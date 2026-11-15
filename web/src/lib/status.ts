import type { BookStatus, HistoryEventType, QueueStatus } from './api'

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
