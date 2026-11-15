import {
  Activity,
  BookOpen,
  Layers,
  LogOut,
  Monitor,
  Moon,
  Plus,
  Server,
  Settings,
  Sun,
  Target,
  Users,
  type LucideIcon,
} from 'lucide-react'
import { NavLink } from 'react-router-dom'
import { IconButton } from '../components/Button'
import { cn } from '../lib/cn'
import { useLogout, useUsername } from '../lib/auth'
import { useStatus } from '../lib/queries'
import { useTheme, type ThemePreference } from '../lib/theme'
import { Brand } from './Brand'

interface NavItem {
  to: string
  label: string
  icon: LucideIcon
  end?: boolean
  badge?: 'queue'
}

// Grouped by what the user is doing: browsing the library, getting more of
// it, and running the app.
const NAV: { label: string; items: NavItem[] }[] = [
  {
    label: 'Browse',
    items: [
      { to: '/', label: 'Library', icon: BookOpen, end: true },
      { to: '/authors', label: 'Authors', icon: Users },
      { to: '/series', label: 'Series', icon: Layers },
    ],
  },
  {
    label: 'Acquire',
    items: [
      { to: '/add', label: 'Add New', icon: Plus },
      { to: '/wanted', label: 'Wanted', icon: Target },
      { to: '/activity', label: 'Activity', icon: Activity, badge: 'queue' },
    ],
  },
  {
    label: 'Manage',
    items: [
      { to: '/settings', label: 'Settings', icon: Settings },
      { to: '/system', label: 'System', icon: Server },
    ],
  },
]

const THEMES: Record<ThemePreference, { icon: LucideIcon; label: string; next: ThemePreference }> = {
  system: { icon: Monitor, label: 'Theme: system. Switch to light', next: 'light' },
  light: { icon: Sun, label: 'Theme: light. Switch to dark', next: 'dark' },
  dark: { icon: Moon, label: 'Theme: dark. Switch to system', next: 'system' },
}

export function SidebarContent({ onNavigate }: { onNavigate?: () => void }) {
  const status = useStatus()
  const username = useUsername()
  const logout = useLogout()
  const [theme, setTheme] = useTheme()
  const queueCount = status.data?.queueCount ?? 0
  const ThemeIcon = THEMES[theme].icon

  return (
    <div className="flex h-full flex-col">
      <div className="flex h-14 items-center px-4">
        <Brand />
      </div>
      <nav className="flex-1 space-y-5 overflow-y-auto px-2 pt-2 pb-4" aria-label="Main">
        {NAV.map((group) => (
          <div key={group.label} role="group" aria-label={group.label}>
            <div className="mb-1 px-2.5 text-2xs font-medium uppercase tracking-wider text-faint">{group.label}</div>
            <div className="space-y-px">
              {group.items.map(({ to, label, icon: Icon, end, badge }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={end}
                  onClick={onNavigate}
                  className={({ isActive }) =>
                    cn(
                      'group flex h-8 items-center gap-2.5 rounded-control px-2.5 text-sm transition-colors',
                      isActive ? 'bg-surface-3 font-medium text-fg' : 'text-muted hover:bg-surface-2 hover:text-fg',
                    )
                  }
                >
                  {({ isActive }) => (
                    <>
                      <Icon
                        className={cn('size-4 shrink-0 transition-colors', isActive ? 'text-accent' : 'text-faint group-hover:text-muted')}
                      />
                      <span className="flex-1 truncate">{label}</span>
                      {badge === 'queue' && queueCount > 0 && (
                        <span
                          className="min-w-5 rounded-chip bg-accent/14 px-1.5 text-center text-xs font-medium text-accent tabular-nums"
                          aria-label={`${queueCount} in queue`}
                        >
                          {queueCount}
                        </span>
                      )}
                    </>
                  )}
                </NavLink>
              ))}
            </div>
          </div>
        ))}
      </nav>
      <div className="flex items-center gap-1 border-t border-line py-2.5 pr-2 pl-3">
        <span className="grid size-7 shrink-0 place-items-center rounded-full bg-surface-3 text-xs font-semibold uppercase text-muted">
          {username.slice(0, 1)}
        </span>
        <div className="ml-1.5 min-w-0 flex-1 leading-tight">
          <div className="truncate text-sm font-medium">{username}</div>
          {status.data && <div className="truncate font-mono text-2xs text-faint">{status.data.version}</div>}
        </div>
        <IconButton label={THEMES[theme].label} size="sm" onClick={() => setTheme(THEMES[theme].next)}>
          <ThemeIcon className="size-4" />
        </IconButton>
        <IconButton label="Sign out" size="sm" loading={logout.isPending} onClick={() => logout.mutate()}>
          <LogOut className="size-4" />
        </IconButton>
      </div>
    </div>
  )
}
