import type { ReactNode } from 'react'
import { Button } from '../../components/Button'
import { Spinner } from '../../components/Spinner'
import { useAuthStatus } from '../../lib/auth'
import { errorMessage } from '../../lib/queryClient'
import { Brand } from '../../layout/Brand'
import { LoginPage } from './LoginPage'
import { SetupPage } from './SetupPage'

/** Renders children only for an authenticated user; otherwise first-run setup or login. */
export function AuthGate({ children }: { children: ReactNode }) {
  const auth = useAuthStatus()

  if (auth.isPending) {
    return (
      <div className="grid min-h-dvh place-items-center bg-canvas">
        <Spinner className="size-5 text-muted" />
      </div>
    )
  }
  if (auth.isError) {
    return (
      <div className="grid min-h-dvh place-items-center bg-canvas px-4">
        <div className="flex flex-col items-center gap-4 text-center">
          <Brand size="lg" />
          <p className="max-w-sm text-sm text-muted">
            Can't reach the Chaptarr server. {errorMessage(auth.error)}
          </p>
          <Button onClick={() => void auth.refetch()} loading={auth.isFetching}>
            Retry
          </Button>
        </div>
      </div>
    )
  }
  if (auth.data.setupRequired) return <SetupPage />
  if (!auth.data.authenticated) return <LoginPage />
  return <>{children}</>
}
