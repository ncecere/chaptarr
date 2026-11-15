import { ArrowUp, Folder } from 'lucide-react'
import { useState } from 'react'
import { Button } from '../../components/Button'
import { Input } from '../../components/Form'
import { Modal } from '../../components/Modal'
import { ErrorState, LoadingState } from '../../components/States'
import { useAddRootFolder } from '../../lib/mutations'
import { useBrowse } from '../../lib/queries'

/** Server-side folder picker; selecting a folder registers it as a root folder. */
export function FolderBrowserModal({ onClose }: { onClose: () => void }) {
  const [path, setPath] = useState<string | undefined>(undefined)
  const [typed, setTyped] = useState<string | null>(null)
  const browse = useBrowse(path)
  const add = useAddRootFolder()
  const listing = browse.data

  const go = (p: string | undefined) => {
    setPath(p)
    setTyped(null)
  }

  return (
    <Modal
      open
      onClose={onClose}
      title="Add root folder"
      busy={add.isPending}
      footer={
        <>
          <span className="mr-auto min-w-0 truncate text-xs text-muted">
            {listing ? listing.path : ''}
          </span>
          <Button onClick={onClose} disabled={add.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            disabled={!listing}
            loading={add.isPending}
            onClick={() => listing && add.mutate(listing.path, { onSuccess: onClose })}
          >
            Select this folder
          </Button>
        </>
      }
    >
      <form
        className="mb-3 flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          if (typed !== null && typed.trim()) go(typed.trim())
        }}
      >
        <Button
          icon={<ArrowUp className="size-4" />}
          disabled={!listing || listing.parent === null}
          onClick={() => listing?.parent != null && go(listing.parent)}
          title="Parent folder"
        >
          Up
        </Button>
        <Input
          value={typed ?? listing?.path ?? path ?? ''}
          onChange={(e) => setTyped(e.target.value)}
          placeholder="/path/to/audiobooks"
          aria-label="Folder path"
          className="font-mono"
          spellCheck={false}
        />
        <Button type="submit" disabled={typed === null}>
          Go
        </Button>
      </form>

      <div className="h-72 overflow-y-auto rounded-control border border-line bg-surface-2/40">
        {browse.isError && !listing ? (
          <ErrorState
            className="m-3 border-0 py-6"
            title="Can't open that folder"
            error={browse.error}
            onRetry={() => go(undefined)}
          />
        ) : !listing ? (
          <LoadingState className="py-10" />
        ) : listing.directories.length === 0 ? (
          <p className="px-4 py-10 text-center text-sm text-muted">No subfolders here.</p>
        ) : (
          <ul className="divide-y divide-line">
            {listing.directories.map((d) => (
              <li key={d.path}>
                <button
                  type="button"
                  onClick={() => go(d.path)}
                  className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm transition-colors hover:bg-surface-3"
                >
                  <Folder className="size-4 shrink-0 text-faint" />
                  <span className="truncate">{d.name}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Modal>
  )
}
