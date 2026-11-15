import { useState } from 'react'
import type { AuthorDetail, MonitorMode } from '../lib/api'
import { useImportAuthorCatalogue } from '../lib/mutations'
import { Button } from './Button'
import { Checkbox } from './Checkbox'
import { Select } from './Form'
import { Modal } from './Modal'
import { Spinner } from './Spinner'

/** Adds every book by an author who is already in the library. */
export function ImportCatalogueModal({
  author,
  open,
  onClose,
}: {
  author: AuthorDetail
  open: boolean
  onClose: () => void
}) {
  const add = useImportAuthorCatalogue()
  const [monitor, setMonitor] = useState<MonitorMode>('all')
  const [monitorNewBooks, setMonitorNewBooks] = useState(true)
  const [searchOnAdd, setSearchOnAdd] = useState(false)

  return (
    <Modal
      open={open}
      onClose={onClose}
      busy={add.isPending}
      size="sm"
      title={`Add all books by ${author.name}`}
      footer={
        <>
          {add.isPending && (
            <span role="status" className="mr-auto flex items-center gap-2 text-sm text-muted">
              <Spinner className="text-accent" />
              Importing…
            </span>
          )}
          <Button variant="ghost" onClick={onClose} disabled={add.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            loading={add.isPending}
            onClick={() =>
              add.mutate(
                {
                  before: author.books.length,
                  request: { asin: author.asin, name: author.name, monitor, monitorNewBooks, searchOnAdd },
                },
                { onSuccess: onClose },
              )
            }
          >
            Add books
          </Button>
        </>
      }
    >
      <div className="space-y-4 text-sm">
        <p className="text-muted">
          Imports the author&rsquo;s catalogue, one entry per work. Books already in your library are kept as they
          are.
        </p>
        <label className="flex items-center justify-between gap-3">
          Monitor the added books
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
          checked={monitorNewBooks && monitor !== 'none'}
          onChange={setMonitorNewBooks}
          disabled={add.isPending || monitor === 'none'}
          label="Add and monitor future releases"
        />
        <Checkbox
          checked={searchOnAdd && monitor !== 'none'}
          onChange={setSearchOnAdd}
          disabled={add.isPending || monitor === 'none'}
          label="Search for missing books now"
        />
      </div>
    </Modal>
  )
}
