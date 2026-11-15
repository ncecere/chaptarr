import { RefreshCw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Badge } from '../../components/Badge'
import { Button } from '../../components/Button'
import { Checkbox } from '../../components/Checkbox'
import { Select } from '../../components/Form'
import { QueryView } from '../../components/QueryView'
import type { LogEntry } from '../../lib/api'
import { cn } from '../../lib/cn'
import { formatDate, formatTime } from '../../lib/format'
import { useLogs } from '../../lib/queries'
import type { Tone } from '../../lib/status'

type Level = LogEntry['level']

const RANK: Record<Level, number> = { debug: 0, info: 1, warn: 2, error: 3 }
const TONE: Record<Level, Tone> = { debug: 'neutral', info: 'info', warn: 'warn', error: 'bad' }
const LIMIT = 300

export function LogViewer() {
  const [minLevel, setMinLevel] = useState<Level>('info')
  const [auto, setAuto] = useState(true)
  const query = useLogs(LIMIT, auto)
  const shown = useMemo(
    () => query.data?.filter((e) => RANK[e.level] >= RANK[minLevel]),
    [query.data, minLevel],
  )

  return (
    <section>
      <div className="mb-3 flex flex-wrap items-center gap-3">
        <h2 className="mr-auto text-base font-semibold">Logs</h2>
        <Checkbox checked={auto} onChange={setAuto} label="Auto-refresh" className="text-muted" />
        <Select
          value={minLevel}
          onChange={(e) => setMinLevel(e.target.value as Level)}
          aria-label="Minimum log level"
          className="h-8 w-auto"
        >
          <option value="debug">Debug and above</option>
          <option value="info">Info and above</option>
          <option value="warn">Warnings and errors</option>
          <option value="error">Errors only</option>
        </Select>
        <Button
          size="sm"
          icon={<RefreshCw className={cn('size-3.5', query.isFetching && 'animate-spin')} />}
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
        >
          Refresh
        </Button>
      </div>
      <QueryView
        query={query}
        data={shown}
        isEmpty={(l) => l.length === 0}
        empty={
          <p className="rounded-xl border border-dashed border-line-strong px-4 py-10 text-center text-sm text-muted">
            No log entries at this level.
          </p>
        }
      >
        {(entries) => (
          <div className="max-h-[32rem] overflow-auto rounded-xl border border-line bg-surface">
            <table className="w-full border-collapse font-mono text-xs">
              <tbody className="divide-y divide-line">
                {entries.map((e, i) => (
                  <tr key={`${e.time}-${i}`} className={cn('align-top', e.level === 'error' && 'bg-bad/5')}>
                    <td className="px-3 py-1.5 whitespace-nowrap text-faint" title={formatDate(e.time)}>
                      {formatTime(e.time)}
                    </td>
                    <td className="px-1 py-1.5">
                      <Badge tone={TONE[e.level]} className="font-sans uppercase">
                        {e.level}
                      </Badge>
                    </td>
                    <td className="px-3 py-1.5 break-words whitespace-pre-wrap">{e.message}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </QueryView>
    </section>
  )
}
