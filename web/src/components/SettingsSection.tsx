import type { ReactNode } from 'react'

export function SettingsSection({
  title,
  description,
  actions,
  children,
}: {
  title: string
  description?: ReactNode
  actions?: ReactNode
  children: ReactNode
}) {
  return (
    <section className="mb-8">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold">{title}</h2>
          {description && <p className="mt-0.5 max-w-2xl text-sm text-muted">{description}</p>}
        </div>
        {actions}
      </div>
      {children}
    </section>
  )
}
