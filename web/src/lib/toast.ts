import { useSyncExternalStore } from 'react'

export type ToastKind = 'success' | 'error' | 'info'

export interface ToastItem {
  id: number
  kind: ToastKind
  message: string
}

// A tiny external store so toasts can be raised from anywhere (including the
// query client's global error handler) without a provider.
let toasts: ToastItem[] = []
let nextId = 1
const listeners = new Set<() => void>()

function emit() {
  for (const l of listeners) l()
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function dismissToast(id: number) {
  toasts = toasts.filter((t) => t.id !== id)
  emit()
}

function push(kind: ToastKind, message: string) {
  const id = nextId++
  toasts = [...toasts, { id, kind, message }].slice(-5)
  emit()
  setTimeout(() => dismissToast(id), kind === 'error' ? 8000 : 4500)
}

export const toast = {
  success: (message: string) => push('success', message),
  error: (message: string) => push('error', message),
  info: (message: string) => push('info', message),
}

export function useToasts(): ToastItem[] {
  return useSyncExternalStore(subscribe, () => toasts)
}
