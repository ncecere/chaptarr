import type { ReactNode } from 'react'
import { QueryView } from '../../components/QueryView'
import { formatBytes, formatDateTime, formatRelative } from '../../lib/format'
import { useStatus } from '../../lib/queries'

function Stat({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="rounded-card border border-line bg-surface px-4 py-3">
      <div className="text-xs text-muted">{label}</div>
      <div className="mt-0.5 truncate text-lg font-semibold tabular-nums">{children}</div>
    </div>
  )
}

export function StatusSummary() {
  const query = useStatus()
  return (
    <section>
      <h2 className="mb-3 text-base font-semibold">Status</h2>
      <QueryView query={query}>
        {(s) => (
          <>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
              <Stat label="Version">{s.version}</Stat>
              <Stat label="Authors">{s.authorCount.toLocaleString()}</Stat>
              <Stat label="Books">{s.bookCount.toLocaleString()}</Stat>
              <Stat label="Downloaded">{s.downloadedCount.toLocaleString()}</Stat>
              <Stat label="Missing">{s.missingCount.toLocaleString()}</Stat>
              <Stat label="In queue">{s.queueCount.toLocaleString()}</Stat>
              <Stat label="Library size">{formatBytes(s.librarySizeBytes)}</Stat>
              <Stat label="Started">
                <span title={formatDateTime(s.startedAt)}>{formatRelative(s.startedAt)}</span>
              </Stat>
            </div>
            <p className="mt-3 text-sm text-muted">
              Data directory: <span className="font-mono text-xs text-fg">{s.dataDir}</span>
            </p>
          </>
        )}
      </QueryView>
    </section>
  )
}
