import { Plus } from 'lucide-react'
import { useState } from 'react'
import type { BookResult, SeriesDetail } from '../lib/api'
import { formatDate, formatRuntime, plural } from '../lib/format'
import { useAddSeriesBooks } from '../lib/mutations'
import { useSeriesMissing } from '../lib/queries'
import { Button } from './Button'
import { Checkbox } from './Checkbox'
import { Cover } from './Cover'
import { Modal } from './Modal'
import { LoadingState } from './States'

/** Lists the series' books that are not in the library and adds the chosen ones. */
export function AddSeriesBooksModal({
  series,
  open,
  onClose,
}: {
  series: SeriesDetail
  open: boolean
  onClose: () => void
}) {
  const missing = useSeriesMissing(series.id, open)
  const add = useAddSeriesBooks()
  // Books the user unticked; everything else is added.
  const [skipped, setSkipped] = useState<ReadonlySet<string>>(new Set())
  const [monitored, setMonitored] = useState(true)
  const [searchOnAdd, setSearchOnAdd] = useState(false)

  const books: BookResult[] = missing.data ?? []
  const chosen = books.filter((b) => !skipped.has(b.asin))
  const toggle = (asin: string, on: boolean) => {
    const next = new Set(skipped)
    if (on) next.delete(asin)
    else next.add(asin)
    setSkipped(next)
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      busy={add.isPending}
      size="lg"
      title={`Add missing books: ${series.name}`}
      footer={
        <>
          <div className="mr-auto flex flex-wrap gap-x-5 gap-y-2">
            <Checkbox checked={monitored} onChange={setMonitored} disabled={add.isPending} label="Monitor" />
            <Checkbox
              checked={searchOnAdd && monitored}
              onChange={setSearchOnAdd}
              disabled={add.isPending || !monitored}
              label="Search for them now"
            />
          </div>
          <Button variant="ghost" onClick={onClose} disabled={add.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            loading={add.isPending}
            disabled={chosen.length === 0}
            onClick={() =>
              add.mutate(
                {
                  id: series.id,
                  request: { asins: chosen.map((b) => b.asin), monitored, searchOnAdd: searchOnAdd && monitored },
                },
                { onSuccess: onClose },
              )
            }
          >
            Add {plural(chosen.length, 'book')}
          </Button>
        </>
      }
    >
      {missing.isPending ? (
        <LoadingState label="Looking up the series…" />
      ) : missing.isError ? (
        <p role="alert" className="py-6 text-center text-sm text-bad">
          {missing.error.message}
        </p>
      ) : books.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted">
          Every book the catalogue lists for this series is already in your library.
        </p>
      ) : (
        <>
          <div className="mb-3 flex items-center justify-between text-sm text-muted">
            <span>
              {plural(books.length, 'book')} not in your library
            </span>
            <Checkbox
              checked={chosen.length === books.length}
              indeterminate={chosen.length > 0}
              onChange={(on) => setSkipped(on ? new Set() : new Set(books.map((b) => b.asin)))}
              disabled={add.isPending}
              label="Select all"
            />
          </div>
          <ul className="divide-y divide-line rounded-lg border border-line">
            {books.map((b) => (
              <li key={b.asin}>
                <label className="flex cursor-pointer items-center gap-3 px-3 py-2.5 hover:bg-surface-2">
                  <input
                    type="checkbox"
                    className="size-4 shrink-0 rounded accent-accent"
                    checked={!skipped.has(b.asin)}
                    disabled={add.isPending}
                    onChange={(e) => toggle(b.asin, e.target.checked)}
                  />
                  <span className="w-8 shrink-0 text-center text-sm tabular-nums text-muted">
                    {b.seriesPosition || '–'}
                  </span>
                  <Cover src={b.coverUrl} alt="" className="size-10 shrink-0" />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium">{b.title}</span>
                    <span className="block truncate text-xs text-muted">
                      {b.authors.map((a) => a.name).join(', ')}
                    </span>
                  </span>
                  <span className="hidden shrink-0 text-right text-xs text-muted sm:block">
                    <span className="block">{formatDate(b.releaseDate)}</span>
                    <span className="block">{b.runtimeMinutes > 0 ? formatRuntime(b.runtimeMinutes) : ''}</span>
                  </span>
                </label>
              </li>
            ))}
          </ul>
        </>
      )}
    </Modal>
  )
}
