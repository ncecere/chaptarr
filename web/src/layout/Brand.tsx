import { Headphones } from 'lucide-react'

export function Brand({ size = 'md' }: { size?: 'md' | 'lg' }) {
  return (
    <div className="flex items-center gap-2.5">
      <span
        className={
          size === 'lg'
            ? 'grid size-11 place-items-center rounded-xl bg-accent text-accent-fg'
            : 'grid size-8 place-items-center rounded-lg bg-accent text-accent-fg'
        }
      >
        <Headphones className={size === 'lg' ? 'size-6' : 'size-5'} strokeWidth={2.25} />
      </span>
      <span className={size === 'lg' ? 'text-2xl font-semibold tracking-tight' : 'text-base font-semibold tracking-tight'}>
        Chaptarr
      </span>
    </div>
  )
}
