import { Headphones } from 'lucide-react'

export function Brand({ size = 'md' }: { size?: 'md' | 'lg' }) {
  return (
    <div className="flex items-center gap-2.5">
      <span
        className={
          size === 'lg'
            ? 'grid size-10 place-items-center rounded-card bg-accent text-accent-fg'
            : 'grid size-7 place-items-center rounded-control bg-accent text-accent-fg'
        }
      >
        <Headphones className={size === 'lg' ? 'size-5.5' : 'size-4'} strokeWidth={2.25} />
      </span>
      <span className={size === 'lg' ? 'text-xl font-semibold tracking-tight' : 'text-[15px] font-semibold tracking-tight'}>
        Chaptarr
      </span>
    </div>
  )
}
