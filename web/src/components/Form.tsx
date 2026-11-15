import { Check } from 'lucide-react'
import { useState, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes } from 'react'
import { cn } from '../lib/cn'

// Sizing, border, focus ring and the select chevron live in index.css (.control).
const control = 'control'

/** Controls fill their container unless the caller sets a width of its own. */
function sized(className?: string | false): string {
  return cn(control, !(className && /(^|\s)w-/.test(className)) && 'w-full', className)
}

/** Label + control + hint. Wraps in a <label>, so it suits a single control. */
export function Field({
  label,
  hint,
  children,
  className,
}: {
  label: string
  hint?: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <label className={cn('block', className)}>
      <span className="mb-1.5 block text-sm font-medium">{label}</span>
      {children}
      {hint && <span className="mt-1.5 block text-xs leading-relaxed text-muted">{hint}</span>}
    </label>
  )
}

/** Like Field, for controls that are not a single input (lists, toggles). */
export function FieldGroup({
  label,
  hint,
  children,
  className,
}: {
  label: string
  hint?: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <div className={className} role="group" aria-label={label}>
      <div className="mb-1.5 text-sm font-medium">{label}</div>
      {children}
      {hint && <div className="mt-1.5 text-xs leading-relaxed text-muted">{hint}</div>}
    </div>
  )
}

export function Input({ className, ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={sized(className)} {...rest} />
}

export function Select({ className, children, ...rest }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select className={sized(className)} {...rest}>
      {children}
    </select>
  )
}

interface NumberInputProps {
  value: number
  onChange: (value: number) => void
  min?: number
  max?: number
  className?: string
}

/** Integer input that lets the user clear the box while typing and snaps back on blur. */
export function NumberInput({ value, onChange, min, max, className }: NumberInputProps) {
  const [text, setText] = useState(String(value))
  const [seen, setSeen] = useState(value)
  if (seen !== value) {
    setSeen(value)
    if (Number(text) !== value) setText(String(value))
  }
  return (
    <input
      type="number"
      inputMode="numeric"
      min={min}
      max={max}
      value={text}
      className={sized(className)}
      onChange={(e) => {
        setText(e.target.value)
        const n = Number(e.target.value)
        if (e.target.value !== '' && Number.isFinite(n)) {
          const clamped = Math.min(max ?? Infinity, Math.max(min ?? -Infinity, Math.trunc(n)))
          setSeen(clamped)
          onChange(clamped)
        }
      }}
      onBlur={() => setText(String(value))}
    />
  )
}

/**
 * Write-only secret field. The server never returns the secret, so when one is
 * stored (`saved`) an empty value means "keep it" and the placeholder says so.
 */
export function SecretInput({
  value,
  onChange,
  saved,
  autoComplete = 'new-password',
}: {
  value: string
  onChange: (value: string) => void
  saved: boolean
  autoComplete?: string
}) {
  return (
    <div className="relative">
      <input
        type="password"
        value={value}
        autoComplete={autoComplete}
        placeholder={saved ? 'Saved — leave blank to keep' : ''}
        onChange={(e) => onChange(e.target.value)}
        className={sized(saved && value === '' && 'pr-20')}
      />
      {saved && value === '' && (
        <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center gap-1 text-xs text-ok">
          <Check className="size-3.5" />
          Saved
        </span>
      )}
    </div>
  )
}
