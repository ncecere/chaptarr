import { cn } from '../lib/cn'

interface ToggleProps {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: string
  disabled?: boolean
  size?: 'sm' | 'md'
  className?: string
}

/** Switch. `label` is the accessible name; use LabeledToggle for visible text. */
export function Toggle({ checked, onChange, label, disabled, size = 'md', className }: ToggleProps) {
  const small = size === 'sm'
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      title={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        'relative inline-flex shrink-0 items-center rounded-full transition-colors',
        'disabled:cursor-not-allowed disabled:opacity-50',
        checked ? 'bg-accent' : 'bg-surface-3 ring-1 ring-inset ring-line-strong',
        small ? 'h-4.5 w-8' : 'h-5.5 w-10',
        className,
      )}
    >
      <span
        className={cn(
          'inline-block rounded-full bg-white shadow transition-transform',
          small ? 'size-3.5' : 'size-4.5',
          checked ? (small ? 'translate-x-4' : 'translate-x-5') : 'translate-x-0.5',
        )}
      />
    </button>
  )
}

/** A Toggle with a visible label beside it. */
export function LabeledToggle({
  label,
  hint,
  ...rest
}: ToggleProps & { label: string; hint?: string }) {
  return (
    <div className="flex items-center gap-3">
      <Toggle label={label} {...rest} />
      <div className="leading-tight">
        <div className="text-sm">{label}</div>
        {hint && <div className="text-xs text-muted">{hint}</div>}
      </div>
    </div>
  )
}
