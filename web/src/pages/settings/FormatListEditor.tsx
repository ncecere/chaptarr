import { ArrowDown, ArrowUp, Plus, X } from 'lucide-react'
import { useState } from 'react'
import { Button, IconButton } from '../../components/Button'
import { Select } from '../../components/Form'

const KNOWN_FORMATS = ['m4b', 'mp3', 'm4a', 'flac', 'ogg', 'opus', 'aac', 'wma', 'mp4']

/** Ordered list of audio formats, most preferred first. */
export function FormatListEditor({
  value,
  onChange,
}: {
  value: string[]
  onChange: (value: string[]) => void
}) {
  const available = KNOWN_FORMATS.filter((f) => !value.includes(f))
  const [toAdd, setToAdd] = useState('')
  const choice = available.includes(toAdd) ? toAdd : (available[0] ?? '')

  const move = (from: number, to: number) => {
    const next = [...value]
    const [item] = next.splice(from, 1)
    next.splice(to, 0, item)
    onChange(next)
  }

  return (
    <div className="max-w-sm space-y-2">
      {value.length === 0 ? (
        <p className="rounded-md border border-dashed border-line-strong px-3 py-3 text-sm text-muted">
          No preference — every format is treated equally.
        </p>
      ) : (
        <ol className="divide-y divide-line overflow-hidden rounded-md border border-line-strong bg-surface-2">
          {value.map((f, i) => (
            <li key={f} className="flex items-center gap-2 py-1.5 pr-1.5 pl-3">
              <span className="w-5 text-xs text-faint tabular-nums">{i + 1}</span>
              <span className="flex-1 text-sm font-medium uppercase">{f}</span>
              <IconButton label={`Move ${f} up`} size="sm" disabled={i === 0} onClick={() => move(i, i - 1)}>
                <ArrowUp className="size-4" />
              </IconButton>
              <IconButton
                label={`Move ${f} down`}
                size="sm"
                disabled={i === value.length - 1}
                onClick={() => move(i, i + 1)}
              >
                <ArrowDown className="size-4" />
              </IconButton>
              <IconButton
                label={`Remove ${f}`}
                size="sm"
                className="hover:text-bad"
                onClick={() => onChange(value.filter((x) => x !== f))}
              >
                <X className="size-4" />
              </IconButton>
            </li>
          ))}
        </ol>
      )}
      {available.length > 0 && (
        <div className="flex gap-2">
          <Select
            value={choice}
            onChange={(e) => setToAdd(e.target.value)}
            aria-label="Format to add"
            className="h-8 flex-1"
          >
            {available.map((f) => (
              <option key={f} value={f}>
                {f.toUpperCase()}
              </option>
            ))}
          </Select>
          <Button size="sm" icon={<Plus className="size-3.5" />} onClick={() => onChange([...value, choice])}>
            Add
          </Button>
        </div>
      )}
    </div>
  )
}
