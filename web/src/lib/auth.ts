import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type AuthStatus, type Credentials } from './api'
import { AUTH_KEY } from './queryClient'

export function useAuthStatus() {
  return useQuery({
    queryKey: AUTH_KEY,
    queryFn: api.authStatus,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    retry: false,
  })
}

/** The signed-in username. Only call below the auth gate. */
export function useUsername(): string {
  const { data } = useAuthStatus()
  return data?.user?.username ?? ''
}

// Login/setup errors are shown inline by their forms, hence meta.silent.
export function useLogin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (c: Credentials) => api.login(c),
    onSuccess: (status) => qc.setQueryData<AuthStatus>(AUTH_KEY, status),
    meta: { silent: true },
  })
}

export function useSetup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (c: Credentials) => api.setup(c),
    onSuccess: (status) => qc.setQueryData<AuthStatus>(AUTH_KEY, status),
    meta: { silent: true },
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.logout(),
    onSettled: () => {
      qc.removeQueries({ predicate: (q) => q.queryKey[0] !== AUTH_KEY[0] })
      qc.setQueryData<AuthStatus>(AUTH_KEY, { setupRequired: false, authenticated: false, user: null })
    },
  })
}
