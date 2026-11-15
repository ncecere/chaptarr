import { Pencil, PlugZap, Trash2 } from 'lucide-react'
import type { ReactNode } from 'react'
import type { TestResult } from '../lib/api'
import { IconButton, Button } from './Button'
import { TestResultNote } from './TestResultNote'
import { Toggle } from './Toggle'

/** One configured indexer or download client in a settings list. */
export function ServiceRow({
  name,
  badges,
  subtitle,
  enabled,
  onToggleEnabled,
  togglePending,
  onTest,
  testing,
  testResult,
  onEdit,
  onDelete,
}: {
  name: string
  badges?: ReactNode
  subtitle?: ReactNode
  enabled: boolean
  onToggleEnabled: (enabled: boolean) => void
  togglePending?: boolean
  onTest: () => void
  testing?: boolean
  testResult?: TestResult
  onEdit: () => void
  onDelete: () => void
}) {
  return (
    <li className="p-4">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-3">
        <div className="min-w-0 flex-1 basis-56">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{name}</span>
            {badges}
          </div>
          {subtitle && <div className="mt-0.5 truncate text-xs text-muted">{subtitle}</div>}
        </div>
        <div className="flex items-center gap-2">
          <Toggle
            checked={enabled}
            disabled={togglePending}
            label={enabled ? `Disable ${name}` : `Enable ${name}`}
            onChange={onToggleEnabled}
          />
          <Button size="sm" icon={<PlugZap className="size-3.5" />} loading={testing} onClick={onTest}>
            Test
          </Button>
          <IconButton label={`Edit ${name}`} size="sm" onClick={onEdit}>
            <Pencil className="size-4" />
          </IconButton>
          <IconButton label={`Delete ${name}`} size="sm" onClick={onDelete} className="hover:text-bad">
            <Trash2 className="size-4" />
          </IconButton>
        </div>
      </div>
      {testResult && <TestResultNote result={testResult} className="mt-3" />}
    </li>
  )
}
