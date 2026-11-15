import { Check, ListChecks, X } from 'lucide-react'
import { Link } from 'react-router-dom'
import { IconButton } from '../components/Button'
import { useLocalStorage } from '../lib/hooks'
import { useStatus } from '../lib/queries'

/** First-run checklist: shown until a root folder, an indexer and a download client exist. */
export function SetupBanner() {
  const { data } = useStatus()
  const [dismissed, setDismissed] = useLocalStorage('setupBannerDismissed', '')
  if (!data) return null

  const steps = [
    { done: data.hasRootFolder, label: 'Add a root folder', to: '/settings/rootfolders' },
    { done: data.hasIndexer, label: 'Add an indexer', to: '/settings/indexers' },
    { done: data.hasDownloadClient, label: 'Add a download client', to: '/settings/downloadclients' },
  ]
  const todo = steps.filter((s) => !s.done)
  // Re-show if the set of outstanding steps changes (e.g. a client is later removed).
  const signature = todo.map((s) => s.to).join(',')
  if (todo.length === 0 || dismissed === signature) return null

  return (
    <section
      aria-label="Setup checklist"
      className="mb-6 rounded-card border border-accent/25 bg-accent/8 p-4"
    >
      <div className="flex items-start gap-3">
        <ListChecks className="mt-0.5 size-4 shrink-0 text-accent" />
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-semibold">Finish setting up Chaptarr</h2>
          <p className="mt-0.5 text-sm text-muted">
            Chaptarr needs a place to store books, somewhere to search, and something to download with.
          </p>
          <ul className="mt-3 flex flex-wrap gap-x-5 gap-y-1.5 text-sm">
            {steps.map((s) => (
              <li key={s.to} className="flex items-center gap-2">
                <span
                  className={
                    s.done
                      ? 'grid size-4 place-items-center rounded-full bg-ok text-on-status'
                      : 'size-4 rounded-full border border-line-strong'
                  }
                  aria-hidden
                >
                  {s.done && <Check className="size-3" strokeWidth={3} />}
                </span>
                {s.done ? (
                  <span className="text-muted line-through">{s.label}</span>
                ) : (
                  <Link to={s.to} className="link">
                    {s.label}
                  </Link>
                )}
              </li>
            ))}
          </ul>
        </div>
        <IconButton label="Dismiss" size="sm" onClick={() => setDismissed(signature)}>
          <X className="size-4" />
        </IconButton>
      </div>
    </section>
  )
}
