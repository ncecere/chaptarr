import { Outlet } from 'react-router-dom'
import { PageHeader } from '../../components/PageHeader'
import { TabNav } from '../../components/Tabs'
import { useDocumentTitle } from '../../lib/hooks'

export function SettingsLayout() {
  useDocumentTitle('Settings')
  return (
    <>
      <PageHeader title="Settings" />
      <TabNav
        items={[
          { to: '/settings/general', label: 'General' },
          { to: '/settings/rootfolders', label: 'Root Folders' },
          { to: '/settings/indexers', label: 'Indexers' },
          { to: '/settings/downloadclients', label: 'Download Clients' },
          { to: '/settings/account', label: 'Account' },
        ]}
      />
      <Outlet />
    </>
  )
}
