import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query'
import { ApiError, type AuthStatus } from './api'
import { toast } from './toast'

export const AUTH_KEY = ['auth'] as const

export function errorMessage(err: unknown): string {
  if (err instanceof Error && err.message) return err.message
  return 'Something went wrong'
}

/**
 * A 401 while signed in means the session is gone: drop every cached
 * response and flip the auth state so the gate renders the login form.
 * Returns true when the error was a 401 (handled here, no toast needed).
 */
function handleUnauthorized(err: unknown): boolean {
  if (!(err instanceof ApiError) || err.status !== 401) return false
  const current = queryClient.getQueryData<AuthStatus>(AUTH_KEY)
  if (current?.authenticated) {
    queryClient.removeQueries({ predicate: (q) => q.queryKey[0] !== AUTH_KEY[0] })
    queryClient.setQueryData<AuthStatus>(AUTH_KEY, { ...current, authenticated: false, user: null })
    toast.info('Your session has expired. Please sign in again.')
  }
  return true
}

export const queryClient: QueryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (err) => {
      handleUnauthorized(err)
    },
  }),
  mutationCache: new MutationCache({
    onError: (err, _vars, _ctx, mutation) => {
      if (handleUnauthorized(err)) return
      // Forms that show the failure inline opt out with meta: { silent: true }.
      if (mutation.meta?.silent === true) return
      toast.error(errorMessage(err))
    },
  }),
  defaultOptions: {
    queries: {
      staleTime: 15_000,
      refetchOnWindowFocus: true,
      retry: (count, err) => {
        if (err instanceof ApiError && err.status < 500) return false
        return count < 2
      },
    },
  },
})
