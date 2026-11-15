import type { ReactNode } from 'react'
import { Button } from './Button'

/**
 * One titled group of settings. On wide screens the title and its
 * explanation sit beside the fields, so a long form reads as a few named
 * groups instead of one column of inputs.
 */
export function FormSection({
  title,
  description,
  children,
}: {
  title: string
  description?: ReactNode
  children: ReactNode
}) {
  return (
    <section className="grid gap-x-12 gap-y-5 border-t border-line py-8 first:border-t-0 first:pt-0 lg:grid-cols-[15rem_minmax(0,1fr)]">
      <div>
        <h2 className="text-base font-semibold">{title}</h2>
        {description && <p className="mt-1 text-sm leading-relaxed text-muted">{description}</p>}
      </div>
      <div className="max-w-xl space-y-6">{children}</div>
    </section>
  )
}

/**
 * Save and discard. The bar appears, pinned to the bottom of the viewport,
 * once the form differs from what is saved, and leaves when it no longer does.
 */
export function FormActions({
  dirty,
  saving,
  onDiscard,
}: {
  dirty: boolean
  saving: boolean
  onDiscard: () => void
}) {
  if (!dirty && !saving) return null
  return (
    <div
      role="region"
      aria-label="Unsaved changes"
      className="sticky bottom-0 z-10 -mx-4 mt-2 flex items-center gap-2 border-t border-line bg-canvas/85 px-4 py-3 backdrop-blur-md animate-toast-in sm:-mx-6 sm:px-6 lg:-mx-10 lg:px-10"
    >
      <span className="mr-auto text-sm text-muted">You have unsaved changes</span>
      <Button variant="ghost" disabled={saving} onClick={onDiscard}>
        Discard
      </Button>
      <Button type="submit" variant="primary" loading={saving}>
        Save changes
      </Button>
    </div>
  )
}
