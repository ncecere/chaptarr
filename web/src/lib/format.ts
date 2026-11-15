// Shared display formatting. Sizes are bytes, runtimes are minutes, dates are
// RFC 3339 (or YYYY-MM-DD for release dates).

const DASH = '—'

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return DASH
  if (bytes < 1024) return `${Math.round(bytes)} B`
  const units = ['KB', 'MB', 'GB', 'TB', 'PB']
  let v = bytes / 1024
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  const digits = v >= 100 ? 0 : v >= 10 ? 1 : 2
  return `${v.toFixed(digits)} ${units[i]}`
}

/** 970 -> "16h 10m", 45 -> "45m", 0 -> "—". */
export function formatRuntime(minutes: number): string {
  if (!Number.isFinite(minutes) || minutes <= 0) return DASH
  const total = Math.round(minutes)
  const h = Math.floor(total / 60)
  const m = total % 60
  if (h === 0) return `${m}m`
  return m === 0 ? `${h}h` : `${h}h ${m}m`
}

const dateFmt = new Intl.DateTimeFormat(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
const dateTimeFmt = new Intl.DateTimeFormat(undefined, {
  year: 'numeric',
  month: 'short',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})
const timeFmt = new Intl.DateTimeFormat(undefined, {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
})

function parseDate(value: string): Date | null {
  if (!value) return null
  // Date-only strings are calendar dates; avoid the UTC shift of `new Date("2024-01-02")`.
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  const d = m ? new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : new Date(value)
  return Number.isNaN(d.getTime()) ? null : d
}

export function formatDate(value: string | null | undefined): string {
  const d = value ? parseDate(value) : null
  return d ? dateFmt.format(d) : DASH
}

export function formatDateTime(value: string | null | undefined): string {
  const d = value ? parseDate(value) : null
  return d ? dateTimeFmt.format(d) : DASH
}

export function formatTime(value: string): string {
  const d = parseDate(value)
  return d ? timeFmt.format(d) : value
}

const relFmt = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })

/** "5 minutes ago", "in 2 hours". */
export function formatRelative(value: string | null | undefined): string {
  const d = value ? parseDate(value) : null
  if (!d) return DASH
  const diffSec = Math.round((d.getTime() - Date.now()) / 1000)
  const abs = Math.abs(diffSec)
  if (abs < 45) return relFmt.format(diffSec, 'second')
  if (abs < 45 * 60) return relFmt.format(Math.round(diffSec / 60), 'minute')
  if (abs < 22 * 3600) return relFmt.format(Math.round(diffSec / 3600), 'hour')
  if (abs < 26 * 86400) return relFmt.format(Math.round(diffSec / 86400), 'day')
  if (abs < 320 * 86400) return relFmt.format(Math.round(diffSec / (30 * 86400)), 'month')
  return relFmt.format(Math.round(diffSec / (365 * 86400)), 'year')
}

/** Seconds remaining -> "3m 20s", "1h 05m". */
export function formatEta(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined || !Number.isFinite(seconds) || seconds <= 0) return DASH
  const s = Math.round(seconds)
  if (s < 60) return `${s}s`
  if (s < 3600) return `${Math.floor(s / 60)}m ${String(s % 60).padStart(2, '0')}s`
  if (s < 86400) return `${Math.floor(s / 3600)}h ${String(Math.floor((s % 3600) / 60)).padStart(2, '0')}m`
  return `${Math.floor(s / 86400)}d ${Math.floor((s % 86400) / 3600)}h`
}

/** Post age in days -> "5h", "3d", "4mo", "1.2y". */
export function formatAge(days: number): string {
  if (!Number.isFinite(days) || days < 0) return DASH
  if (days < 1) return `${Math.max(1, Math.round(days * 24))}h`
  if (days < 30) return `${Math.round(days)}d`
  if (days < 365) return `${Math.round(days / 30)}mo`
  return `${(days / 365).toFixed(1)}y`
}

export function formatDurationMs(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return DASH
  if (ms < 1000) return `${Math.round(ms)}ms`
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.floor(ms / 60_000)}m ${Math.round((ms % 60_000) / 1000)}s`
}

/** Task interval in seconds -> "Every 15 min". 0 means on demand. */
export function formatInterval(seconds: number): string {
  if (!seconds || seconds <= 0) return 'On demand'
  if (seconds < 60) return `Every ${seconds}s`
  if (seconds < 3600) return `Every ${Math.round(seconds / 60)} min`
  if (seconds < 86400) return `Every ${+(seconds / 3600).toFixed(1)} h`
  return `Every ${+(seconds / 86400).toFixed(1)} d`
}

export function formatPercent(fraction: number): string {
  return `${Math.round(Math.min(1, Math.max(0, fraction)) * 100)}%`
}

export function releaseYear(date: string): string {
  const m = /^(\d{4})/.exec(date)
  return m ? m[1] : ''
}

export function plural(n: number, singular: string, pluralForm = `${singular}s`): string {
  return `${n.toLocaleString()} ${n === 1 ? singular : pluralForm}`
}

/** Title-case a camelCase identifier: "downloadFailed" -> "Download failed". */
export function humanize(s: string): string {
  const spaced = s.replace(/([a-z])([A-Z])/g, '$1 $2').toLowerCase()
  return spaced.charAt(0).toUpperCase() + spaced.slice(1)
}
