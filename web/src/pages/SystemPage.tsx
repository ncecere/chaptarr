import { PageHeader } from '../components/PageHeader'
import { useDocumentTitle } from '../lib/hooks'
import { LogViewer } from './system/LogViewer'
import { StatusSummary } from './system/StatusSummary'
import { TasksTable } from './system/TasksTable'

export function SystemPage() {
  useDocumentTitle('System')
  return (
    <>
      <PageHeader title="System" />
      <div className="space-y-10">
        <StatusSummary />
        <TasksTable />
        <LogViewer />
      </div>
    </>
  )
}
