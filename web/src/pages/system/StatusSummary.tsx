import type { ReactNode } from 'react'
import { QueryView } from '../../components/QueryView'
import { formatBytes, formatDateTime, formatRelative } from '../../lib/format'
import { useStatus } from '../../lib/queries'

function Stat({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="bg-surface px-4 py-3.5">
      <dt className="text-2xs font-medium uppercase tracking-wider text-faint">{label}</dt>
      <dd className="mt-1 truncate text-xl font-semibold tracking-tight tabular-nums">{children}</dd>
    </div>
  )
}

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 items-baseline gap-2">
      <dt className="shrink-0 text-muted">{label}</dt>
      <dd className="min-w-0 truncate">{children}</dd>
    </div>
  )
}

export function StatusSummary() {
  const query = useStatus()
  return (
    <section aria-labelledby="system-status">
      <h2 id="system-status" className="mb-3 text-base font-semibold">
        Status
      </h2>
      <QueryView query={query}>
        {(s) => (
          <div className="overflow-hidden rounded-card border border-line bg-surface">
            {/* The 1px gaps show the line colour behind the cells, giving hairline dividers at any column count. */}
            <dl className="grid grid-cols-2 gap-px bg-line sm:grid-cols-3 lg:grid-cols-6">
              <Stat label="Authors">{s.authorCount.toLocaleString()}</Stat>
              <Stat label="Books">{s.bookCount.toLocaleString()}</Stat>
              <Stat label="Downloaded">{s.downloadedCount.toLocaleString()}</Stat>
              <Stat label="Missing">{s.missingCount.toLocaleString()}</Stat>
              <Stat label="In queue">{s.queueCount.toLocaleString()}</Stat>
              <Stat label="Library size">{formatBytes(s.librarySizeBytes)}</Stat>
            </dl>
            <dl className="flex flex-wrap gap-x-8 gap-y-1.5 border-t border-line px-4 py-3 text-sm">
              <Fact label="Version">
                <span className="font-mono text-xs">{s.version}</span>
              </Fact>
              <Fact label="Started">
                <span title={formatDateTime(s.startedAt)}>{formatRelative(s.startedAt)}</span>
              </Fact>
              <Fact label="Data directory">
                <span className="font-mono text-xs">{s.dataDir}</span>
              </Fact>
            </dl>
          </div>
        )}
      </QueryView>
    </section>
  )
}
