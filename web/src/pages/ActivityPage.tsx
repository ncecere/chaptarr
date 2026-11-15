import { useSearchParams } from 'react-router-dom'
import { PageHeader } from '../components/PageHeader'
import { StateTabs } from '../components/Tabs'
import { useDocumentTitle } from '../lib/hooks'
import { useStatus } from '../lib/queries'
import { HistoryTab } from './activity/HistoryTab'
import { QueueTab } from './activity/QueueTab'

type Tab = 'queue' | 'history'

export function ActivityPage() {
  useDocumentTitle('Activity')
  const [params, setParams] = useSearchParams()
  const tab: Tab = params.get('tab') === 'history' ? 'history' : 'queue'
  const queueCount = useStatus().data?.queueCount ?? 0

  return (
    <>
      <PageHeader title="Activity" />
      <StateTabs<Tab>
        value={tab}
        onChange={(t) => setParams(t === 'queue' ? {} : { tab: t }, { replace: true })}
        tabs={[
          {
            value: 'queue',
            label: (
              <>
                Queue
                {queueCount > 0 && (
                  <span className="rounded-full bg-accent px-1.5 text-xs font-semibold text-accent-fg tabular-nums">
                    {queueCount}
                  </span>
                )}
              </>
            ),
          },
          { value: 'history', label: 'History' },
        ]}
      />
      {tab === 'queue' ? <QueueTab /> : <HistoryTab />}
    </>
  )
}
