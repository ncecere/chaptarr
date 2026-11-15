import {
  Activity,
  BookOpen,
  Layers,
  LogOut,
  Plus,
  Server,
  Settings,
  Target,
  Users,
  type LucideIcon,
} from 'lucide-react'
import { NavLink } from 'react-router-dom'
import { IconButton } from '../components/Button'
import { cn } from '../lib/cn'
import { useLogout, useUsername } from '../lib/auth'
import { useStatus } from '../lib/queries'
import { Brand } from './Brand'

interface NavItem {
  to: string
  label: string
  icon: LucideIcon
  end?: boolean
  badge?: 'queue'
}

const NAV: NavItem[] = [
  { to: '/', label: 'Library', icon: BookOpen, end: true },
  { to: '/authors', label: 'Authors', icon: Users },
  { to: '/series', label: 'Series', icon: Layers },
  { to: '/add', label: 'Add New', icon: Plus },
  { to: '/wanted', label: 'Wanted', icon: Target },
  { to: '/activity', label: 'Activity', icon: Activity, badge: 'queue' },
  { to: '/settings', label: 'Settings', icon: Settings },
  { to: '/system', label: 'System', icon: Server },
]

export function SidebarContent({ onNavigate }: { onNavigate?: () => void }) {
  const status = useStatus()
  const username = useUsername()
  const logout = useLogout()
  const queueCount = status.data?.queueCount ?? 0

  return (
    <div className="flex h-full flex-col">
      <div className="px-4 py-4">
        <Brand />
      </div>
      <nav className="flex-1 space-y-0.5 overflow-y-auto px-2 py-1" aria-label="Main">
        {NAV.map(({ to, label, icon: Icon, end, badge }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            onClick={onNavigate}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
                isActive
                  ? 'bg-accent/12 font-medium text-accent'
                  : 'text-muted hover:bg-surface-2 hover:text-fg',
              )
            }
          >
            <Icon className="size-4.5" />
            <span className="flex-1">{label}</span>
            {badge === 'queue' && queueCount > 0 && (
              <span
                className="min-w-5 rounded-full bg-accent px-1.5 text-center text-xs font-semibold text-accent-fg tabular-nums"
                aria-label={`${queueCount} in queue`}
              >
                {queueCount}
              </span>
            )}
          </NavLink>
        ))}
      </nav>
      <div className="flex items-center gap-2 border-t border-line px-3 py-3">
        <span className="grid size-8 shrink-0 place-items-center rounded-full bg-surface-3 text-xs font-semibold uppercase">
          {username.slice(0, 1)}
        </span>
        <div className="min-w-0 flex-1 leading-tight">
          <div className="truncate text-sm font-medium">{username}</div>
          {status.data && <div className="truncate text-xs text-faint">{status.data.version}</div>}
        </div>
        <IconButton label="Sign out" loading={logout.isPending} onClick={() => logout.mutate()}>
          <LogOut className="size-4" />
        </IconButton>
      </div>
    </div>
  )
}
