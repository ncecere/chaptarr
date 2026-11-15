import { FolderOpen, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '../../components/Badge'
import { Button, IconButton } from '../../components/Button'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { ProgressBar } from '../../components/ProgressBar'
import { QueryView } from '../../components/QueryView'
import { SettingsSection } from '../../components/SettingsSection'
import { EmptyState } from '../../components/States'
import type { RootFolder } from '../../lib/api'
import { formatBytes } from '../../lib/format'
import { useDeleteRootFolder } from '../../lib/mutations'
import { useRootFolders, useSettings } from '../../lib/queries'
import { FolderBrowserModal } from './FolderBrowserModal'

export function RootFoldersSettings() {
  const query = useRootFolders()
  const settings = useSettings()
  const del = useDeleteRootFolder()
  const [adding, setAdding] = useState(false)
  const [deleting, setDeleting] = useState<RootFolder | null>(null)
  const defaultId = settings.data?.defaultRootFolderId ?? null

  const addButton = (
    <Button variant="primary" icon={<Plus className="size-4" />} onClick={() => setAdding(true)}>
      Add Root Folder
    </Button>
  )

  return (
    <SettingsSection
      title="Root folders"
      description="Folders Chaptarr imports finished audiobooks into."
      actions={addButton}
    >
      <QueryView
        query={query}
        isEmpty={(f) => f.length === 0}
        empty={
          <EmptyState
            icon={<FolderOpen />}
            title="No root folders"
            description="Add the folder where your audiobook library lives."
            action={addButton}
          />
        }
      >
        {(folders) => (
          <ul className="divide-y divide-line overflow-hidden rounded-card border border-line bg-surface">
            {folders.map((f) => {
              const used = f.totalBytes > 0 ? 1 - f.freeBytes / f.totalBytes : 0
              return (
                <li key={f.id} className="flex items-center gap-4 p-4">
                  <FolderOpen className="size-4 shrink-0 text-muted" />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-sm break-all">{f.path}</span>
                      {f.id === defaultId && <Badge tone="accent">Default</Badge>}
                      {!f.accessible && (
                        <Badge tone="bad" title="The directory is missing or not writable">
                          Not accessible
                        </Badge>
                      )}
                    </div>
                    {f.accessible && f.totalBytes > 0 && (
                      <div className="mt-2 max-w-sm">
                        <ProgressBar label="Disk space used" value={used} tone={used > 0.9 ? 'warn' : 'accent'} />
                        <div className="mt-1 text-xs text-muted">
                          {formatBytes(f.freeBytes)} free of {formatBytes(f.totalBytes)}
                        </div>
                      </div>
                    )}
                  </div>
                  <IconButton label={`Remove ${f.path}`} className="hover:text-bad" onClick={() => setDeleting(f)}>
                    <Trash2 className="size-4" />
                  </IconButton>
                </li>
              )
            })}
          </ul>
        )}
      </QueryView>

      {adding && <FolderBrowserModal onClose={() => setAdding(false)} />}

      <ConfirmDialog
        open={deleting !== null}
        title="Remove root folder?"
        danger
        confirmLabel="Remove"
        message={
          <>
            Chaptarr will stop using <span className="font-mono text-fg">{deleting?.path}</span>. Files already on
            disk are not touched.
          </>
        }
        pending={del.isPending}
        onCancel={() => setDeleting(null)}
        onConfirm={() => deleting && del.mutate(deleting.id, { onSettled: () => setDeleting(null) })}
      />
    </SettingsSection>
  )
}
