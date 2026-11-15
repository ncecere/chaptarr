import { useId, useState } from 'react'
import { Button } from '../../components/Button'
import { Field, Input, NumberInput, SecretInput, Select } from '../../components/Form'
import { Modal } from '../../components/Modal'
import { TestResultNote } from '../../components/TestResultNote'
import { LabeledToggle } from '../../components/Toggle'
import type { Indexer, IndexerInput, IndexerType, TestResult } from '../../lib/api'
import { useSaveIndexer, useTestIndexer } from '../../lib/mutations'
import { errorMessage } from '../../lib/queryClient'
import { toIndexerInput } from '../../lib/serviceInputs'

const EMPTY: IndexerInput = {
  name: '',
  type: 'torznab',
  url: '',
  apiKey: '',
  categories: '3030',
  enabled: true,
  priority: 25,
}

export function IndexerModal({ indexer, onClose }: { indexer?: Indexer; onClose: () => void }) {
  const formId = useId()
  const [form, setForm] = useState<IndexerInput>(indexer ? toIndexerInput(indexer) : EMPTY)
  const save = useSaveIndexer()
  const test = useTestIndexer()

  const patch = (p: Partial<IndexerInput>) => {
    setForm((f) => ({ ...f, ...p }))
    if (test.data || test.isError) test.reset()
  }
  const result: TestResult | undefined =
    test.data ?? (test.isError ? { ok: false, message: errorMessage(test.error) } : undefined)

  return (
    <Modal
      open
      onClose={onClose}
      busy={save.isPending}
      title={indexer ? `Edit ${indexer.name}` : 'Add indexer'}
      footer={
        <>
          <Button
            className="mr-auto"
            loading={test.isPending}
            disabled={!form.url}
            onClick={() => test.mutate({ input: form, id: indexer?.id })}
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
          save.mutate({ id: indexer?.id, input: form }, { onSuccess: onClose })
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Name">
            <Input value={form.name} onChange={(e) => patch({ name: e.target.value })} required autoFocus />
          </Field>
          <Field label="Type">
            <Select value={form.type} onChange={(e) => patch({ type: e.target.value as IndexerType })}>
              <option value="torznab">Torznab (torrent)</option>
              <option value="newznab">Newznab (usenet)</option>
            </Select>
          </Field>
        </div>
        <Field label="URL" hint="Base URL of the feed, e.g. http://prowlarr:9696/1/api">
          <Input
            type="url"
            value={form.url}
            onChange={(e) => patch({ url: e.target.value })}
            placeholder="http://prowlarr:9696/1/api"
            required
          />
        </Field>
        <Field label="API key">
          <SecretInput
            value={form.apiKey}
            onChange={(apiKey) => patch({ apiKey })}
            saved={!!indexer?.hasApiKey}
          />
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Categories" hint="Comma-separated newznab category ids.">
            <Input value={form.categories} onChange={(e) => patch({ categories: e.target.value })} />
          </Field>
          <Field label="Priority" hint="1 (highest) to 50. Breaks score ties.">
            <NumberInput value={form.priority} min={1} max={50} onChange={(priority) => patch({ priority })} />
          </Field>
        </div>
        <LabeledToggle label="Enabled" checked={form.enabled} onChange={(enabled) => patch({ enabled })} />
        {result && <TestResultNote result={result} />}
      </form>
    </Modal>
  )
}
