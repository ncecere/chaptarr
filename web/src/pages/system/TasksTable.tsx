import { useQuery } from '@tanstack/react-query'
import { Play } from 'lucide-react'
import { useRef } from 'react'
import { Badge } from '../../components/Badge'
import { Button } from '../../components/Button'
import { QueryView } from '../../components/QueryView'
import { Spinner } from '../../components/Spinner'
import { api } from '../../lib/api'
import { formatDateTime, formatDurationMs, formatInterval, formatRelative } from '../../lib/format'
import { useRunTask } from '../../lib/mutations'

const th = 'th'

export function TasksTable() {
  // Right after Run is pressed the task may not report running yet, so keep polling briefly.
  const pollUntil = useRef(0)
  const query = useQuery({
    queryKey: ['tasks'],
    queryFn: api.tasks,
    refetchInterval: (q) =>
      q.state.data?.some((t) => t.running) || Date.now() < pollUntil.current ? 2_000 : 15_000,
  })
  const run = useRunTask()

  return (
    <section>
      <h2 className="mb-3 text-base font-semibold">Tasks</h2>
      <QueryView query={query}>
        {(tasks) => (
          <div className="overflow-hidden rounded-card border border-line bg-surface">
            <ul className="divide-y divide-line sm:hidden">
              {tasks.map((t) => (
                <li key={t.name} className="flex items-center gap-3 px-3 py-2.5">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2 text-sm font-medium">
                      {t.label}
                      {t.running && (
                        <Badge tone="accent">
                          <Spinner className="size-3" />
                          Running
                        </Badge>
                      )}
                    </div>
                    <div className="text-xs text-muted">
                      {formatInterval(t.intervalSeconds)} · last {t.lastRunAt ? formatRelative(t.lastRunAt) : 'never'}
                      {t.nextRunAt && ` · next ${formatRelative(t.nextRunAt)}`}
                    </div>
                    {t.lastError && <div className="mt-0.5 break-words text-xs text-bad">{t.lastError}</div>}
                  </div>
                <Button
                  size="sm"
                  icon={<Play className="size-3.5" />}
                  disabled={t.running}
                  loading={run.isPending && run.variables.name === t.name}
                  onClick={() =>
                    run.mutate(
                      { name: t.name, label: t.label },
                      { onSuccess: () => (pollUntil.current = Date.now() + 8_000) },
                    )
                  }
                >
                  Run
                </Button>
                </li>
              ))}
            </ul>
            <div className="hidden overflow-x-auto sm:block">
            <table className="w-full min-w-[44rem] border-collapse text-sm">
              <thead>
                <tr className="border-b border-line bg-surface-2/50">
                  <th className={th}>Task</th>
                  <th className={th}>Schedule</th>
                  <th className={th}>Last run</th>
                  <th className={th}>Duration</th>
                  <th className={th}>Next run</th>
                  <th className="w-28"><span className="sr-only">Run</span></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {tasks.map((t) => (
                  <tr key={t.name} className="align-top">
                    <td className="px-3 py-2.5">
                      <div className="flex items-center gap-2 font-medium">
                        {t.label}
                        {t.running && (
                          <Badge tone="accent">
                            <Spinner className="size-3" />
                            Running
                          </Badge>
                        )}
                      </div>
                      {t.lastError && <div className="mt-0.5 break-words text-xs text-bad">{t.lastError}</div>}
                    </td>
                    <td className="px-3 py-2.5 whitespace-nowrap text-muted">{formatInterval(t.intervalSeconds)}</td>
                    <td className="px-3 py-2.5 whitespace-nowrap text-muted" title={formatDateTime(t.lastRunAt)}>
                      {t.lastRunAt ? formatRelative(t.lastRunAt) : 'Never'}
                    </td>
                    <td className="px-3 py-2.5 whitespace-nowrap text-muted tabular-nums">
                      {t.lastRunAt ? formatDurationMs(t.lastDurationMs) : '—'}
                    </td>
                    <td className="px-3 py-2.5 whitespace-nowrap text-muted" title={formatDateTime(t.nextRunAt)}>
                      {t.nextRunAt ? formatRelative(t.nextRunAt) : '—'}
                    </td>
                    <td className="px-3 py-2 text-right">
                      <Button
                        size="sm"
                        icon={<Play className="size-3.5" />}
                        disabled={t.running}
                        loading={run.isPending && run.variables.name === t.name}
                        onClick={() =>
                          run.mutate(
                            { name: t.name, label: t.label },
                            { onSuccess: () => (pollUntil.current = Date.now() + 8_000) },
                          )
                        }
                      >
                        Run
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            </div>
          </div>
        )}
      </QueryView>
    </section>
  )
}
