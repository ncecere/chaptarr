import { ArrowUp, Folder, FolderInput } from 'lucide-react'
import { useState } from 'react'
import type { Settings } from '../lib/api'
import { formatBytes, plural } from '../lib/format'
import { useManualImport } from '../lib/mutations'
import { useBrowse, useInspectImport, useSettings } from '../lib/queries'
import { errorMessage } from '../lib/queryClient'
import { Button } from './Button'
import { Input, Select } from './Form'
import { Modal } from './Modal'
import { Spinner } from './Spinner'

const MODES: { value: Settings['importMode']; label: string }[] = [
  { value: 'move', label: 'Move the files' },
  { value: 'copy', label: 'Copy the files' },
  { value: 'hardlink', label: 'Hardlink the files' },
]

/**
 * Imports the audio in a folder the user picks into one book. With a queue
 * item it starts in that download's folder and settles the item on success.
 */
export function ManualImportModal({
  book,
  queueId = 0,
  onClose,
}: {
  book: { id: number; title: string }
  queueId?: number
  onClose: () => void
}) {
  const settings = useSettings()
  const run = useManualImport()
  // '' means "wherever the download client saved the queue item".
  const [path, setPath] = useState('')
  const [typed, setTyped] = useState<string | null>(null)
  const [mode, setMode] = useState<Settings['importMode'] | null>(null)

  const inspect = useInspectImport(path, queueId, path !== '' || queueId > 0)
  const current = path || inspect.data?.path || ''
  const browse = useBrowse(current || undefined)
  const listing = browse.data
  const contents = inspect.data
  const chosenMode = mode ?? settings.data?.importMode ?? 'copy'

  const go = (p: string) => {
    setPath(p)
    setTyped(null)
    if (run.isError) run.reset()
  }

  return (
    <Modal
      open
      onClose={onClose}
      busy={run.isPending}
      size="lg"
      title={`Manual import: ${book.title}`}
      footer={
        <>
          <label className="mr-auto flex items-center gap-2 text-sm">
            <Select
              value={chosenMode}
              onChange={(e) => setMode(e.target.value as Settings['importMode'])}
              disabled={run.isPending}
              className="h-9 w-auto"
              aria-label="How to import"
            >
              {MODES.map((m) => (
                <option key={m.value} value={m.value}>
                  {m.label}
                </option>
              ))}
            </Select>
          </label>
          <Button variant="ghost" onClick={onClose} disabled={run.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            icon={<FolderInput className="size-4" />}
            loading={run.isPending}
            disabled={!contents || contents.files.length === 0}
            onClick={() =>
              contents &&
              run.mutate(
                { bookId: book.id, request: { path: contents.path, mode: chosenMode, queueId } },
                { onSuccess: onClose },
              )
            }
          >
            Import {contents && contents.files.length > 0 ? plural(contents.files.length, 'file') : ''}
          </Button>
        </>
      }
    >
      <p className="mb-3 text-sm text-muted">
        Pick the folder that holds this book&rsquo;s audio. Every audio file in it, including subfolders, is filed
        under the book.
      </p>
      <form
        className="mb-3 flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          if (typed !== null && typed.trim()) go(typed.trim())
        }}
      >
        <Button
          icon={<ArrowUp className="size-4" />}
          disabled={!listing || listing.parent === null || run.isPending}
          onClick={() => listing?.parent != null && go(listing.parent)}
          title="Parent folder"
        >
          Up
        </Button>
        <Input
          value={typed ?? current}
          onChange={(e) => setTyped(e.target.value)}
          placeholder="/path/to/the/download"
          aria-label="Folder path"
          className="font-mono"
          spellCheck={false}
          disabled={run.isPending}
        />
        <Button type="submit" disabled={typed === null || run.isPending}>
          Go
        </Button>
      </form>

      <div className="grid gap-3 md:grid-cols-2">
        <div>
          <div className="mb-1 text-xs font-medium text-muted">Subfolders</div>
          <div className="h-56 overflow-y-auto rounded-lg border border-line bg-surface-2/40">
            {!listing ? (
              <p className="px-4 py-8 text-center text-sm text-muted">
                {browse.isError ? 'This is not a folder that can be browsed.' : 'Loading…'}
              </p>
            ) : listing.directories.length === 0 ? (
              <p className="px-4 py-8 text-center text-sm text-muted">No subfolders here.</p>
            ) : (
              <ul className="divide-y divide-line">
                {listing.directories.map((d) => (
                  <li key={d.path}>
                    <button
                      type="button"
                      disabled={run.isPending}
                      onClick={() => go(d.path)}
                      className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm transition-colors hover:bg-surface-3"
                    >
                      <Folder className="size-4 shrink-0 text-accent" />
                      <span className="truncate">{d.name}</span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>

        <div>
          <div className="mb-1 text-xs font-medium text-muted">
            {contents && contents.files.length > 0
              ? `${plural(contents.files.length, 'audio file')} · ${formatBytes(contents.totalBytes)}`
              : 'Audio in this folder'}
          </div>
          <div className="h-56 overflow-y-auto rounded-lg border border-line bg-surface-2/40">
            {inspect.isFetching ? (
              <p className="flex items-center justify-center gap-2 px-4 py-8 text-sm text-muted">
                <Spinner className="text-accent" /> Looking for audio…
              </p>
            ) : inspect.isError ? (
              <p role="alert" className="px-4 py-8 text-center text-sm text-bad">
                {errorMessage(inspect.error)}
              </p>
            ) : !contents ? (
              <p className="px-4 py-8 text-center text-sm text-muted">Open a folder to see what it holds.</p>
            ) : contents.files.length === 0 ? (
              <p className="px-4 py-8 text-center text-sm text-muted">
                No audio files here{contents.other ? `: ${contents.other}` : ''}.
              </p>
            ) : (
              <ul className="divide-y divide-line">
                {contents.files.map((f) => (
                  <li key={f.name} className="flex items-baseline gap-3 px-3 py-1.5 text-xs">
                    <span className="min-w-0 flex-1 truncate font-mono" title={f.name}>
                      {f.name}
                    </span>
                    <span className="shrink-0 tabular-nums text-muted">{formatBytes(f.sizeBytes)}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </div>

      {run.isError && (
        <p role="alert" className="mt-3 break-words text-sm text-bad">
          {errorMessage(run.error)}
        </p>
      )}
    </Modal>
  )
}
