import { Plus, Rss } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '../../components/Badge'
import { Button } from '../../components/Button'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { QueryView } from '../../components/QueryView'
import { ServiceRow } from '../../components/ServiceRow'
import { SettingsSection } from '../../components/SettingsSection'
import { EmptyState } from '../../components/States'
import type { Indexer, TestResult } from '../../lib/api'
import { useDeleteIndexer, useSaveIndexer, useTestIndexer } from '../../lib/mutations'
import { errorMessage } from '../../lib/queryClient'
import { useIndexers } from '../../lib/queries'
import { toIndexerInput } from '../../lib/serviceInputs'
import { IndexerModal } from './IndexerModal'

function IndexerRow({
  indexer,
  onEdit,
  onDelete,
}: {
  indexer: Indexer
  onEdit: () => void
  onDelete: () => void
}) {
  const test = useTestIndexer()
  const save = useSaveIndexer()
  const result: TestResult | undefined =
    test.data ?? (test.isError ? { ok: false, message: errorMessage(test.error) } : undefined)

  return (
    <ServiceRow
      name={indexer.name}
      badges={
        <>
          <Badge tone={indexer.type === 'torznab' ? 'info' : 'neutral'}>
            {indexer.type === 'torznab' ? 'Torznab' : 'Newznab'}
          </Badge>
          <Badge>Priority {indexer.priority}</Badge>
        </>
      }
      subtitle={indexer.url}
      enabled={indexer.enabled}
      togglePending={save.isPending}
      onToggleEnabled={(enabled) =>
        save.mutate({
          id: indexer.id,
          input: { ...toIndexerInput(indexer), enabled },
          silentSuccess: true,
        })
      }
      onTest={() => test.mutate({ input: toIndexerInput(indexer), id: indexer.id })}
      testing={test.isPending}
      testResult={result}
      onEdit={onEdit}
      onDelete={onDelete}
    />
  )
}

export function IndexersSettings() {
  const query = useIndexers()
  const del = useDeleteIndexer()
  // `null` closed, `'new'` adding, otherwise the indexer being edited.
  const [editing, setEditing] = useState<Indexer | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Indexer | null>(null)

  const addButton = (
    <Button variant="primary" icon={<Plus className="size-4" />} onClick={() => setEditing('new')}>
      Add Indexer
    </Button>
  )

  return (
    <SettingsSection
      title="Indexers"
      description="Torznab and Newznab feeds Chaptarr searches for releases. Prowlarr works well as a single source."
      actions={addButton}
    >
      <QueryView
        query={query}
        isEmpty={(i) => i.length === 0}
        empty={
          <EmptyState
            icon={<Rss />}
            title="No indexers"
            description="Add at least one indexer so Chaptarr can find audiobooks."
            action={addButton}
          />
        }
      >
        {(indexers) => (
          <ul className="divide-y divide-line overflow-hidden rounded-card border border-line bg-surface">
            {indexers.map((i) => (
              <IndexerRow key={i.id} indexer={i} onEdit={() => setEditing(i)} onDelete={() => setDeleting(i)} />
            ))}
          </ul>
        )}
      </QueryView>

      {editing !== null && (
        <IndexerModal indexer={editing === 'new' ? undefined : editing} onClose={() => setEditing(null)} />
      )}

      <ConfirmDialog
        open={deleting !== null}
        title="Delete indexer?"
        danger
        confirmLabel="Delete"
        message={
          <>
            <span className="font-medium text-fg">{deleting?.name}</span> will no longer be searched.
          </>
        }
        pending={del.isPending}
        onCancel={() => setDeleting(null)}
        onConfirm={() => deleting && del.mutate(deleting.id, { onSettled: () => setDeleting(null) })}
      />
    </SettingsSection>
  )
}
