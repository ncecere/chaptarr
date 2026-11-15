import { useState } from 'react'
import { Button } from '../../components/Button'
import { Field, FieldGroup, Input, NumberInput, Select } from '../../components/Form'
import { QueryView } from '../../components/QueryView'
import { LabeledToggle } from '../../components/Toggle'
import type { RootFolder, Settings } from '../../lib/api'
import { useUpdateSettings } from '../../lib/mutations'
import { useRootFolders, useSettings } from '../../lib/queries'
import { FormatListEditor } from './FormatListEditor'

const IMPORT_MODES: { value: Settings['importMode']; label: string; hint: string }[] = [
  {
    value: 'hardlink',
    label: 'Hardlink',
    hint: 'Keeps seeding without using extra space. Falls back to copy across filesystems.',
  },
  { value: 'copy', label: 'Copy', hint: 'Duplicates the files; the download stays untouched.' },
  { value: 'move', label: 'Move', hint: 'Moves the files out of the download folder.' },
]

const TOKENS = ['{Author}', '{Series}', '{SeriesPosition}', '{Title}', '{Year}', '{Narrator}', '{ASIN}']

function GeneralForm({ settings, folders }: { settings: Settings; folders: RootFolder[] }) {
  const save = useUpdateSettings()
  const [draft, setDraft] = useState<Settings>(settings)
  const patch = (p: Partial<Settings>) => setDraft((d) => ({ ...d, ...p }))
  const dirty = JSON.stringify(draft) !== JSON.stringify(settings)
  const mode = IMPORT_MODES.find((m) => m.value === draft.importMode)

  return (
    <form
      className="max-w-2xl space-y-6"
      onSubmit={(e) => {
        e.preventDefault()
        save.mutate(draft)
      }}
    >
      <div>
        <Field
          label="Folder format"
          hint="Path inside the root folder. Segments that end up empty (like {Series} for a standalone) are dropped."
        >
          <Input
            value={draft.folderFormat}
            onChange={(e) => patch({ folderFormat: e.target.value })}
            className="font-mono"
            spellCheck={false}
          />
        </Field>
        <div className="mt-2 flex flex-wrap gap-1.5">
          {TOKENS.map((t) => (
            <button
              key={t}
              type="button"
              onClick={() => patch({ folderFormat: draft.folderFormat + t })}
              className="rounded border border-line-strong bg-surface-2 px-1.5 py-0.5 font-mono text-xs text-muted transition-colors hover:border-accent hover:text-fg"
            >
              {t}
            </button>
          ))}
        </div>
      </div>

      <FieldGroup label="Preferred formats" hint="Most preferred first. Automatic search scores releases by this order.">
        <FormatListEditor value={draft.preferredFormats} onChange={(preferredFormats) => patch({ preferredFormats })} />
      </FieldGroup>

      <Field label="Import mode" hint={mode?.hint}>
        <Select
          value={draft.importMode}
          onChange={(e) => patch({ importMode: e.target.value as Settings['importMode'] })}
          className="max-w-xs"
        >
          {IMPORT_MODES.map((m) => (
            <option key={m.value} value={m.value}>
              {m.label}
            </option>
          ))}
        </Select>
      </Field>

      <LabeledToggle
        label="Remove completed downloads"
        hint="After importing, clear usenet downloads out of the download client's history. Torrents are left to seed."
        checked={draft.removeCompletedDownloads}
        onChange={(removeCompletedDownloads) => patch({ removeCompletedDownloads })}
      />

      <Field label="Default root folder" hint="New books are imported here.">
        <Select
          value={draft.defaultRootFolderId ?? ''}
          onChange={(e) => patch({ defaultRootFolderId: e.target.value === '' ? null : Number(e.target.value) })}
          className="max-w-md"
        >
          <option value="">{folders.length === 0 ? 'No root folders yet' : 'None'}</option>
          {folders.map((f) => (
            <option key={f.id} value={f.id}>
              {f.path}
            </option>
          ))}
        </Select>
      </Field>

      <div className="grid gap-6 sm:grid-cols-3">
        <Field label="Minimum seeders" hint="Torrents below this are skipped by automatic search.">
          <NumberInput value={draft.minSeeders} min={0} onChange={(minSeeders) => patch({ minSeeders })} />
        </Field>
        <Field label="Missing search interval" hint="Minutes between automatic searches. 0 disables.">
          <NumberInput
            value={draft.missingSearchIntervalMinutes}
            min={0}
            onChange={(missingSearchIntervalMinutes) => patch({ missingSearchIntervalMinutes })}
          />
        </Field>
        <Field label="Author refresh interval" hint="Hours between metadata refreshes. 0 disables.">
          <NumberInput
            value={draft.authorRefreshIntervalHours}
            min={0}
            onChange={(authorRefreshIntervalHours) => patch({ authorRefreshIntervalHours })}
          />
        </Field>
      </div>

      <div className="flex gap-2 border-t border-line pt-5">
        <Button type="submit" variant="primary" loading={save.isPending} disabled={!dirty}>
          Save changes
        </Button>
        <Button variant="ghost" disabled={!dirty || save.isPending} onClick={() => setDraft(settings)}>
          Discard
        </Button>
      </div>
    </form>
  )
}

export function GeneralSettings() {
  const settings = useSettings()
  const folders = useRootFolders()
  return (
    <QueryView query={settings}>
      {(s) => (
        // Re-key on the saved values so the form resets cleanly after a save.
        <GeneralForm key={JSON.stringify(s)} settings={s} folders={folders.data ?? []} />
      )}
    </QueryView>
  )
}
