import { CircleAlert, CircleCheck } from 'lucide-react'
import type { TestResult } from '../lib/api'
import { cn } from '../lib/cn'

export function TestResultNote({ result, className }: { result: TestResult; className?: string }) {
  return (
    <div
      role="status"
      className={cn(
        'flex items-start gap-2 rounded-control border px-3 py-2 text-sm',
        result.ok ? 'border-ok/30 bg-ok/8 text-ok' : 'border-bad/30 bg-bad/8 text-bad',
        className,
      )}
    >
      {result.ok ? (
        <CircleCheck className="mt-0.5 size-4 shrink-0" />
      ) : (
        <CircleAlert className="mt-0.5 size-4 shrink-0" />
      )}
      <span className="min-w-0 break-words">
        {result.message || (result.ok ? 'Connection successful' : 'Test failed')}
      </span>
    </div>
  )
}
