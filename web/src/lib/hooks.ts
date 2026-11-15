import { useCallback, useEffect, useState } from 'react'

export function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delayMs)
    return () => clearTimeout(t)
  }, [value, delayMs])
  return debounced
}

export function useLocalStorage<T>(key: string, initial: T): [T, (value: T) => void] {
  const storageKey = `chaptarr.${key}`
  const [value, setValue] = useState<T>(() => {
    try {
      const raw = localStorage.getItem(storageKey)
      return raw === null ? initial : (JSON.parse(raw) as T)
    } catch {
      return initial
    }
  })
  const set = useCallback(
    (next: T) => {
      setValue(next)
      try {
        localStorage.setItem(storageKey, JSON.stringify(next))
      } catch {
        // Storage may be unavailable (private mode); the preference just won't persist.
      }
    },
    [storageKey],
  )
  return [value, set]
}

export function useDocumentTitle(title?: string) {
  useEffect(() => {
    document.title = title ? `${title} · Chaptarr` : 'Chaptarr'
  }, [title])
}
