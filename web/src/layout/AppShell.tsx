import { Menu } from 'lucide-react'
import { useState } from 'react'
import { Outlet } from 'react-router-dom'
import { IconButton } from '../components/Button'
import { Brand } from './Brand'
import { SetupBanner } from './SetupBanner'
import { SidebarContent } from './Sidebar'

export function AppShell() {
  const [drawerOpen, setDrawerOpen] = useState(false)
  return (
    <div className="min-h-dvh lg:pl-60">
      {/* Desktop sidebar */}
      <aside className="fixed inset-y-0 left-0 z-30 hidden w-60 border-r border-line bg-surface lg:block">
        <SidebarContent />
      </aside>

      {/* Mobile top bar + drawer */}
      <header className="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-line bg-surface/95 px-3 backdrop-blur lg:hidden">
        <IconButton label="Open menu" onClick={() => setDrawerOpen(true)}>
          <Menu className="size-5" />
        </IconButton>
        <Brand />
      </header>
      {drawerOpen && (
        <div className="fixed inset-0 z-40 lg:hidden">
          <div
            className="absolute inset-0 bg-black/70 animate-fade-in"
            onClick={() => setDrawerOpen(false)}
            aria-hidden
          />
          <aside className="absolute inset-y-0 left-0 w-64 max-w-[85vw] border-r border-line bg-surface shadow-2xl">
            <SidebarContent onNavigate={() => setDrawerOpen(false)} />
          </aside>
        </div>
      )}

      <main className="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
        <SetupBanner />
        <Outlet />
      </main>
    </div>
  )
}
