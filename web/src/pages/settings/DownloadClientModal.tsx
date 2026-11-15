import { useId, useState } from 'react'
import { Button } from '../../components/Button'
import { Field, Input, SecretInput, Select } from '../../components/Form'
import { Modal } from '../../components/Modal'
import { TestResultNote } from '../../components/TestResultNote'
import { LabeledToggle } from '../../components/Toggle'
import type { DownloadClient, DownloadClientInput, DownloadClientType, TestResult } from '../../lib/api'
import { useSaveDownloadClient, useTestDownloadClient } from '../../lib/mutations'
import { errorMessage } from '../../lib/queryClient'
import { toClientInput } from '../../lib/serviceInputs'

const EMPTY: DownloadClientInput = {
  name: '',
  type: 'qbittorrent',
  url: '',
  username: '',
  password: '',
  apiKey: '',
  category: 'audiobooks',
  enabled: true,
}

/** Drops the fields that do not apply to the selected client type. */
function normalize(f: DownloadClientInput): DownloadClientInput {
  return f.type === 'qbittorrent' ? { ...f, apiKey: '' } : { ...f, username: '', password: '' }
}

export function DownloadClientModal({ client, onClose }: { client?: DownloadClient; onClose: () => void }) {
  const formId = useId()
  const [form, setForm] = useState<DownloadClientInput>(client ? toClientInput(client) : EMPTY)
  const save = useSaveDownloadClient()
  const test = useTestDownloadClient()

  const patch = (p: Partial<DownloadClientInput>) => {
    setForm((f) => ({ ...f, ...p }))
    if (test.data || test.isError) test.reset()
  }
  const result: TestResult | undefined =
    test.data ?? (test.isError ? { ok: false, message: errorMessage(test.error) } : undefined)

  // A stored secret only applies while the type is unchanged.
  const sameType = client?.type === form.type

  return (
    <Modal
      open
      onClose={onClose}
      busy={save.isPending}
      title={client ? `Edit ${client.name}` : 'Add download client'}
      footer={
        <>
          <Button
            className="mr-auto"
            loading={test.isPending}
            disabled={!form.url}
            onClick={() => test.mutate({ input: normalize(form), id: client?.id })}
          >
            Test
          </Button>
          <Button variant="ghost" onClick={onClose} disabled={save.isPending}>
            Cancel
          </Button>
          <Button type="submit" form={formId} variant="primary" loading={save.isPending}>
            Save
          </Button>
        </>
      }
    >
      <form
        id={formId}
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault()
          save.mutate({ id: client?.id, input: normalize(form) }, { onSuccess: onClose })
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Name">
            <Input value={form.name} onChange={(e) => patch({ name: e.target.value })} required autoFocus />
          </Field>
          <Field label="Type">
            <Select
              value={form.type}
              onChange={(e) => patch({ type: e.target.value as DownloadClientType })}
            >
              <option value="qbittorrent">qBittorrent (torrent)</option>
              <option value="sabnzbd">SABnzbd (usenet)</option>
            </Select>
          </Field>
        </div>
        <Field label="URL" hint="Base URL of the client's web interface.">
          <Input
            type="url"
            value={form.url}
            onChange={(e) => patch({ url: e.target.value })}
            placeholder="http://localhost:8080"
            required
          />
        </Field>
        {form.type === 'qbittorrent' ? (
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Username">
              <Input
                value={form.username}
                onChange={(e) => patch({ username: e.target.value })}
                autoComplete="off"
              />
            </Field>
            <Field label="Password">
              <SecretInput
                value={form.password}
                onChange={(password) => patch({ password })}
                saved={!!client?.hasPassword && sameType}
              />
            </Field>
          </div>
        ) : (
          <Field label="API key">
            <SecretInput
              value={form.apiKey}
              onChange={(apiKey) => patch({ apiKey })}
              saved={!!client?.hasApiKey && sameType}
            />
          </Field>
        )}
        <Field label="Category" hint="Category or label applied to downloads.">
          <Input value={form.category} onChange={(e) => patch({ category: e.target.value })} />
        </Field>
        <LabeledToggle label="Enabled" checked={form.enabled} onChange={(enabled) => patch({ enabled })} />
        {result && <TestResultNote result={result} />}
      </form>
    </Modal>
  )
}
