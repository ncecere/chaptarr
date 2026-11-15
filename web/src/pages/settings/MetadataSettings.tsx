import { ArrowDown, ArrowUp } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '../../components/Badge'
import { Button, IconButton } from '../../components/Button'
import { Field, Input, SecretInput, Select } from '../../components/Form'
import { FormActions } from '../../components/FormLayout'
import { QueryView } from '../../components/QueryView'
import { TestResultNote } from '../../components/TestResultNote'
import { Toggle } from '../../components/Toggle'
import type { MetadataSource, MetadataSourceType, Settings, TestResult } from '../../lib/api'
import { useTestMetadataSource, useUpdateSettings } from '../../lib/mutations'
import { useSettings } from '../../lib/queries'
import { errorMessage } from '../../lib/queryClient'
import { metadataSourceMeta } from '../../lib/status'

const REGIONS: { value: string; label: string }[] = [
  { value: 'us', label: 'United States (us)' },
  { value: 'uk', label: 'United Kingdom (uk)' },
  { value: 'ca', label: 'Canada (ca)' },
  { value: 'au', label: 'Australia (au)' },
  { value: 'de', label: 'Germany (de)' },
  { value: 'fr', label: 'France (fr)' },
  { value: 'es', label: 'Spain (es)' },
  { value: 'it', label: 'Italy (it)' },
  { value: 'in', label: 'India (in)' },
  { value: 'jp', label: 'Japan (jp)' },
]

const SOURCES: Record<MetadataSourceType, { about: string; defaultUrl: string; needsKey?: boolean }> = {
  audible: {
    about:
      'The primary catalogue. Authors, series, narrators and runtimes come only from here, so it is always on.',
    defaultUrl: 'https://api.audible.com',
  },
  itunes: {
    about: 'Apple’s audiobook store. Adds audiobooks Audible does not list, with descriptions and large covers.',
    defaultUrl: 'https://itunes.apple.com',
  },
  openlibrary: {
    about:
      'The Internet Archive’s book catalogue. Covers books that have no audiobook listing anywhere else; it describes the printed book.',
    defaultUrl: 'https://openlibrary.org',
  },
  googlebooks: {
    about: 'Google’s book catalogue, mainly for descriptions and covers. Needs a Google Books API key.',
    defaultUrl: 'https://www.googleapis.com/books/v1',
    needsKey: true,
  },
}

function SourceCard({
  source,
  position,
  count,
  disabled,
  onChange,
  onMove,
  children,
}: {
  source: MetadataSource
  position: number
  count: number
  disabled: boolean
  onChange: (patch: Partial<MetadataSource>) => void
  onMove: (delta: number) => void
  children?: React.ReactNode
}) {
  const info = SOURCES[source.type]
  const primary = source.type === 'audible'
  const test = useTestMetadataSource()
  const result: TestResult | undefined =
    test.data ?? (test.isError ? { ok: false, message: errorMessage(test.error) } : undefined)
  const change = (patch: Partial<MetadataSource>) => {
    if (test.data || test.isError) test.reset()
    onChange(patch)
  }
  const missingKey = info.needsKey && !source.hasApiKey && !source.apiKey

  return (
    <li className="rounded-card border border-line bg-surface p-4">
      <div className="flex items-start gap-3">
        <span className="mt-px grid size-5 shrink-0 place-items-center rounded-chip bg-surface-3 text-2xs font-medium tabular-nums text-muted">
          {position + 1}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="font-medium">{metadataSourceMeta[source.type].label}</h3>
            {primary && (
              <Badge tone="accent">Primary</Badge>
            )}
          </div>
          <p className="mt-0.5 text-sm text-muted">{info.about}</p>
        </div>
        {!primary && (
          <div className="flex shrink-0 items-center gap-1">
            <IconButton label="Move up" size="sm" disabled={disabled || position <= 1} onClick={() => onMove(-1)}>
              <ArrowUp className="size-4" />
            </IconButton>
            <IconButton
              label="Move down"
              size="sm"
              disabled={disabled || position >= count - 1}
              onClick={() => onMove(1)}
            >
              <ArrowDown className="size-4" />
            </IconButton>
            <Toggle
              className="ml-2"
              label={`Use ${metadataSourceMeta[source.type].label}`}
              checked={source.enabled}
              disabled={disabled || (missingKey && !source.enabled)}
              onChange={(enabled) => change({ enabled })}
            />
          </div>
        )}
      </div>

      {(primary || source.enabled || info.needsKey) && (
        <div className="mt-4 space-y-4 border-t border-line pt-4 sm:ml-8">
          {children}
          {info.needsKey && (
            <Field
              label="API key"
              hint="Create one in the Google Cloud console with the Books API enabled. The source can be switched on once a key is entered."
            >
              <SecretInput
                value={source.apiKey ?? ''}
                saved={source.hasApiKey}
                onChange={(apiKey) => change({ apiKey })}
              />
            </Field>
          )}
          <Field label="Server URL" hint="Only for a mirror or a self-hosted server. Leave empty for the public one.">
            <Input
              value={source.url}
              onChange={(e) => change({ url: e.target.value })}
              placeholder={info.defaultUrl}
              spellCheck={false}
              inputMode="url"
            />
          </Field>
          <div className="flex flex-wrap items-center gap-3">
            <Button
              size="sm"
              loading={test.isPending}
              disabled={missingKey}
              onClick={() => test.mutate({ type: source.type, url: source.url, apiKey: source.apiKey ?? '' })}
            >
              Test
            </Button>
            {result && <TestResultNote result={result} className="min-w-0 flex-1" />}
          </div>
        </div>
      )}
    </li>
  )
}

function MetadataForm({ settings }: { settings: Settings }) {
  const save = useUpdateSettings()
  const [draft, setDraft] = useState<Settings>(settings)
  const patch = (p: Partial<Settings>) => setDraft((d) => ({ ...d, ...p }))
  const dirty = JSON.stringify(draft) !== JSON.stringify(settings)
  const sources = draft.metadataSources

  const patchSource = (type: MetadataSourceType, p: Partial<MetadataSource>) =>
    patch({ metadataSources: sources.map((s) => (s.type === type ? { ...s, ...p } : s)) })
  const move = (from: number, delta: number) => {
    const to = from + delta
    if (to < 1 || to >= sources.length) return // Audible stays first
    const next = [...sources]
    const [item] = next.splice(from, 1)
    next.splice(to, 0, item)
    patch({ metadataSources: next })
  }

  // Keep a region the server already has even if it is not in our list.
  const regions = REGIONS.some((r) => r.value === draft.audibleRegion)
    ? REGIONS
    : [...REGIONS, { value: draft.audibleRegion, label: draft.audibleRegion }]

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        save.mutate(draft)
      }}
    >
      <div className="mb-4 max-w-3xl">
        <h2 className="text-base font-semibold">Metadata sources</h2>
        <p className="mt-1 text-sm leading-relaxed text-muted">
          Add New searches every source that is switched on, in this order. When two sources list the same book, the
          higher one is shown and the lower one fills in anything it lacks, such as a description or cover.
        </p>
      </div>

      <ul className="max-w-3xl space-y-3">
        {sources.map((source, i) => (
          <SourceCard
            key={source.type}
            source={source}
            position={i}
            count={sources.length}
            disabled={save.isPending}
            onChange={(p) => patchSource(source.type, p)}
            onMove={(delta) => move(i, delta)}
          >
            {source.type === 'audible' && (
              <>
                <div className="grid gap-4 sm:grid-cols-2">
                  <Field label="Region" hint="The Audible marketplace to read.">
                    <Select value={draft.audibleRegion} onChange={(e) => patch({ audibleRegion: e.target.value })}>
                      {regions.map((r) => (
                        <option key={r.value} value={r.value}>
                          {r.label}
                        </option>
                      ))}
                    </Select>
                  </Field>
                  <Field label="Language" hint="Only books in this language are added automatically. Empty for any.">
                    <Input
                      value={draft.language}
                      onChange={(e) => patch({ language: e.target.value })}
                      placeholder="Any language"
                    />
                  </Field>
                </div>
                <Field
                  label="Author details server (Audnexus)"
                  hint="Author photos and biographies. Leave empty for the public server."
                >
                  <Input
                    value={draft.audnexusUrl}
                    onChange={(e) => patch({ audnexusUrl: e.target.value })}
                    placeholder="https://api.audnex.us"
                    spellCheck={false}
                    inputMode="url"
                  />
                </Field>
              </>
            )}
          </SourceCard>
        ))}
      </ul>

      <div className="mt-6" />
      <FormActions dirty={dirty} saving={save.isPending} onDiscard={() => setDraft(settings)} />
    </form>
  )
}

export function MetadataSettings() {
  const settings = useSettings()
  return (
    <QueryView query={settings}>
      {(s) => (
        // Re-key on the saved values so the form resets cleanly after a save.
        <MetadataForm key={JSON.stringify(s)} settings={s} />
      )}
    </QueryView>
  )
}
