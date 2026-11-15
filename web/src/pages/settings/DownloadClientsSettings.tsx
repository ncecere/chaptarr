import { Download, Plus } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '../../components/Badge'
import { Button } from '../../components/Button'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { QueryView } from '../../components/QueryView'
import { ServiceRow } from '../../components/ServiceRow'
import { SettingsSection } from '../../components/SettingsSection'
import { EmptyState } from '../../components/States'
import type { DownloadClient, TestResult } from '../../lib/api'
import {
  useDeleteDownloadClient,
  useSaveDownloadClient,
  useTestDownloadClient,
} from '../../lib/mutations'
import { useDownloadClients } from '../../lib/queries'
import { errorMessage } from '../../lib/queryClient'
import { toClientInput } from '../../lib/serviceInputs'
import { DownloadClientModal } from './DownloadClientModal'

function ClientRow({
  client,
  onEdit,
  onDelete,
}: {
  client: DownloadClient
  onEdit: () => void
  onDelete: () => void
}) {
  const test = useTestDownloadClient()
  const save = useSaveDownloadClient()
  const result: TestResult | undefined =
    test.data ?? (test.isError ? { ok: false, message: errorMessage(test.error) } : undefined)

  return (
    <ServiceRow
      name={client.name}
      badges={
        <>
          <Badge tone={client.type === 'qbittorrent' ? 'info' : 'neutral'}>
            {client.type === 'qbittorrent' ? 'qBittorrent' : 'SABnzbd'}
          </Badge>
          {client.category && <Badge>{client.category}</Badge>}
        </>
      }
      subtitle={client.url}
      enabled={client.enabled}
      togglePending={save.isPending}
      onToggleEnabled={(enabled) =>
        save.mutate({
          id: client.id,
          input: { ...toClientInput(client), enabled },
          silentSuccess: true,
        })
      }
      onTest={() => test.mutate({ input: toClientInput(client), id: client.id })}
      testing={test.isPending}
      testResult={result}
      onEdit={onEdit}
      onDelete={onDelete}
    />
  )
}

export function DownloadClientsSettings() {
  const query = useDownloadClients()
  const del = useDeleteDownloadClient()
  const [editing, setEditing] = useState<DownloadClient | 'new' | null>(null)
  const [deleting, setDeleting] = useState<DownloadClient | null>(null)

  const addButton = (
    <Button variant="primary" icon={<Plus className="size-4" />} onClick={() => setEditing('new')}>
      Add Download Client
    </Button>
  )

  return (
    <SettingsSection
      title="Download clients"
      description="qBittorrent for torrents and SABnzbd for usenet. Chaptarr sends grabbed releases here."
      actions={addButton}
    >
      <QueryView
        query={query}
        isEmpty={(c) => c.length === 0}
        empty={
          <EmptyState
            icon={<Download />}
            title="No download clients"
            description="Add a client so Chaptarr has somewhere to send releases."
            action={addButton}
          />
        }
      >
        {(clients) => (
          <ul className="divide-y divide-line overflow-hidden rounded-xl border border-line bg-surface">
            {clients.map((c) => (
              <ClientRow key={c.id} client={c} onEdit={() => setEditing(c)} onDelete={() => setDeleting(c)} />
            ))}
          </ul>
        )}
      </QueryView>

      {editing !== null && (
        <DownloadClientModal client={editing === 'new' ? undefined : editing} onClose={() => setEditing(null)} />
      )}

      <ConfirmDialog
        open={deleting !== null}
        title="Delete download client?"
        danger
        confirmLabel="Delete"
        message={
          <>
            <span className="font-medium text-fg">{deleting?.name}</span> will no longer receive downloads.
          </>
        }
        pending={del.isPending}
        onCancel={() => setDeleting(null)}
        onConfirm={() => deleting && del.mutate(deleting.id, { onSettled: () => setDeleting(null) })}
      />
    </SettingsSection>
  )
}
