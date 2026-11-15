import { useEffect, useSyncExternalStore } from 'react'

/** The user's choice. "system" follows the operating system and changes with it. */
export type ThemePreference = 'system' | 'light' | 'dark'

// Plain string, not JSON: the inline script in index.html reads it before React loads.
const KEY = 'chaptarr-theme'
const CHROME = { dark: '#0d0e11', light: '#f6f6f8' }

const listeners = new Set<() => void>()
const media = window.matchMedia('(prefers-color-scheme: light)')

function read(): ThemePreference {
  try {
    const v = localStorage.getItem(KEY)
    return v === 'light' || v === 'dark' ? v : 'system'
  } catch {
    return 'system'
  }
}

function resolve(pref: ThemePreference): 'light' | 'dark' {
  return pref === 'light' || (pref === 'system' && media.matches) ? 'light' : 'dark'
}

function apply() {
  const theme = resolve(read())
  document.documentElement.dataset.theme = theme
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', CHROME[theme])
}

function subscribe(onChange: () => void) {
  listeners.add(onChange)
  return () => {
    listeners.delete(onChange)
  }
}

export function setThemePreference(pref: ThemePreference) {
  try {
    if (pref === 'system') localStorage.removeItem(KEY)
    else localStorage.setItem(KEY, pref)
  } catch {
    // Storage may be unavailable (private mode); the choice then lasts for this page only.
  }
  apply()
  for (const l of listeners) l()
}

/** The theme preference and a setter. Mount once near the root to follow system changes. */
export function useTheme(): [ThemePreference, (pref: ThemePreference) => void] {
  const pref = useSyncExternalStore(subscribe, read)
  useEffect(() => {
    apply()
    media.addEventListener('change', apply)
    return () => media.removeEventListener('change', apply)
  }, [])
  return [pref, setThemePreference]
}
