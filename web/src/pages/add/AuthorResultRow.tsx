import { Check, Plus } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Button } from '../../components/Button'
import { Checkbox } from '../../components/Checkbox'
import { Cover } from '../../components/Cover'
import { LinkButton } from '../../components/LinkButton'
import { Select } from '../../components/Form'
import { Spinner } from '../../components/Spinner'
import type { AuthorResult, MonitorMode } from '../../lib/api'
import { useAddAuthor } from '../../lib/mutations'
import { htmlToText } from '../../lib/text'

export function AuthorResultRow({ author }: { author: AuthorResult }) {
  const add = useAddAuthor()
  const [optionsOpen, setOptionsOpen] = useState(false)
  const [monitor, setMonitor] = useState<MonitorMode>('all')
  const [monitorNewBooks, setMonitorNewBooks] = useState(true)
  const [searchOnAdd, setSearchOnAdd] = useState(false)
  const [addedId, setAddedId] = useState<number | null>(null)
  const description = useMemo(() => htmlToText(author.description), [author.description])

  const libraryId = addedId ?? author.libraryAuthorId

  return (
    <li className="rounded-card border border-line bg-surface p-3 sm:p-4">
      <div className="flex gap-3 sm:gap-4">
        <Cover src={author.imageUrl} alt={author.name} kind="author" className="size-16 sm:size-20" />
        <div className="min-w-0 flex-1">
          <h3 className="font-medium leading-snug">{author.name}</h3>
          {description && <p className="mt-1 line-clamp-2 text-xs text-muted">{description}</p>}
        </div>
        <div className="shrink-0">
          {libraryId !== null ? (
            <LinkButton to={`/authors/${libraryId}`} tone="ok" icon={<Check className="size-4" />}>
              In library
            </LinkButton>
          ) : (
            <Button
              icon={<Plus className="size-4" />}
              onClick={() => setOptionsOpen(!optionsOpen)}
              disabled={add.isPending}
              aria-expanded={optionsOpen}
            >
              Add
            </Button>
          )}
        </div>
      </div>
      {optionsOpen && libraryId === null && (
        <div className="mt-3 space-y-3 rounded-control bg-surface-2 px-4 py-3">
          <div className="flex flex-wrap items-center gap-x-6 gap-y-3">
            <label className="flex items-center gap-2 text-sm">
              Monitor
              <Select
                value={monitor}
                onChange={(e) => setMonitor(e.target.value as MonitorMode)}
                disabled={add.isPending}
                className="h-8 w-auto"
              >
                <option value="all">All books</option>
                <option value="future">Future books only</option>
                <option value="none">None</option>
              </Select>
            </label>
            <Checkbox
              checked={monitorNewBooks}
              onChange={setMonitorNewBooks}
              disabled={add.isPending}
              label="Monitor new books"
            />
            <Checkbox
              checked={searchOnAdd}
              onChange={setSearchOnAdd}
              disabled={add.isPending}
              label="Search for missing books now"
            />
          </div>
          <div className="flex items-center justify-end gap-3">
            {add.isPending && (
              <span role="status" className="mr-auto flex items-center gap-2 text-sm text-muted">
                <Spinner className="text-accent" />
                Importing the catalogue — this can take several seconds…
              </span>
            )}
            <Button size="sm" variant="ghost" onClick={() => setOptionsOpen(false)} disabled={add.isPending}>
              Cancel
            </Button>
            <Button
              size="sm"
              variant="primary"
              loading={add.isPending}
              onClick={() =>
                add.mutate(
                  { asin: author.asin, name: author.name, monitor, monitorNewBooks, searchOnAdd },
                  { onSuccess: (a) => setAddedId(a.id) },
                )
              }
            >
              Add author
            </Button>
          </div>
        </div>
      )}
    </li>
  )
}
