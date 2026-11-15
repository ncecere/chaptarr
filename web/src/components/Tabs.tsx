import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import { cn } from '../lib/cn'

/** Route-backed tab strip. */
export function TabNav({ items }: { items: { to: string; label: string; end?: boolean }[] }) {
  return (
    <nav className="mb-6 flex gap-1 overflow-x-auto border-b border-line" aria-label="Sections">
      {items.map((it) => (
        <NavLink
          key={it.to}
          to={it.to}
          end={it.end}
          className={({ isActive }) =>
            cn(
              '-mb-px whitespace-nowrap border-b-2 px-3.5 py-2.5 text-sm transition-colors',
              isActive
                ? 'border-accent font-medium text-fg'
                : 'border-transparent text-muted hover:text-fg',
            )
          }
        >
          {it.label}
        </NavLink>
      ))}
    </nav>
  )
}

/** State-backed tab strip with the same look as TabNav. */
export function StateTabs<T extends string>({
  tabs,
  value,
  onChange,
}: {
  tabs: { value: T; label: ReactNode }[]
  value: T
  onChange: (value: T) => void
}) {
  return (
    <div className="mb-6 flex gap-1 overflow-x-auto border-b border-line" role="tablist">
      {tabs.map((t) => (
        <button
          key={t.value}
          type="button"
          role="tab"
          aria-selected={value === t.value}
          onClick={() => onChange(t.value)}
          className={cn(
            '-mb-px inline-flex items-center gap-2 whitespace-nowrap border-b-2 px-3.5 py-2.5 text-sm transition-colors',
            value === t.value
              ? 'border-accent font-medium text-fg'
              : 'border-transparent text-muted hover:text-fg',
          )}
        >
          {t.label}
        </button>
      ))}
    </div>
  )
}

export function SegmentedControl<T extends string>({
  options,
  value,
  onChange,
  label,
}: {
  options: { value: T; label: ReactNode; title?: string }[]
  value: T
  onChange: (value: T) => void
  label: string
}) {
  return (
    <div
      role="radiogroup"
      aria-label={label}
      className="inline-flex rounded-md border border-line-strong bg-surface p-0.5"
    >
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          role="radio"
          aria-checked={value === o.value}
          title={o.title}
          onClick={() => onChange(o.value)}
          className={cn(
            'inline-flex h-7 items-center gap-1.5 rounded px-3 text-sm transition-colors',
            value === o.value ? 'bg-surface-3 text-fg' : 'text-muted hover:text-fg',
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}
