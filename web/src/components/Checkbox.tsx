import { useEffect, useRef, type ReactNode } from 'react'
import { cn } from '../lib/cn'

interface CheckboxProps {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: ReactNode
  indeterminate?: boolean
  disabled?: boolean
  className?: string
  'aria-label'?: string
}

export function Checkbox({
  checked,
  onChange,
  label,
  indeterminate,
  disabled,
  className,
  'aria-label': ariaLabel,
}: CheckboxProps) {
  const ref = useRef<HTMLInputElement>(null)
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = !!indeterminate && !checked
  }, [indeterminate, checked])
  return (
    <label className={cn('inline-flex items-center gap-2 text-sm', disabled && 'text-muted', className)}>
      <input
        ref={ref}
        type="checkbox"
        checked={checked}
        disabled={disabled}
        aria-label={ariaLabel}
        onChange={(e) => onChange(e.target.checked)}
        className="checkbox"
      />
      {label}
    </label>
  )
}
