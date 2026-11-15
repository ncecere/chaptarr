import { useState, type FormEvent } from 'react'
import { Button } from '../../components/Button'
import { Field, Input } from '../../components/Form'
import { useSetup } from '../../lib/auth'
import { useDocumentTitle } from '../../lib/hooks'
import { errorMessage } from '../../lib/queryClient'
import { AuthLayout } from './AuthLayout'

export function SetupPage() {
  useDocumentTitle('Setup')
  const setup = useSetup()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')

  const mismatch = confirm !== '' && password !== confirm

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (password !== confirm) return
    setup.mutate({ username: username.trim(), password })
  }

  return (
    <AuthLayout
      title="Create your admin account"
      subtitle="Chaptarr is ready. Choose the login you'll use to manage it."
    >
      <form onSubmit={submit} className="space-y-4">
        <Field label="Username">
          <Input
            autoFocus
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
          />
        </Field>
        <Field label="Password">
          <Input
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </Field>
        <Field label="Confirm password">
          <Input
            type="password"
            autoComplete="new-password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            aria-invalid={mismatch}
            required
          />
        </Field>
        {mismatch && (
          <p role="alert" className="text-sm text-bad">
            Passwords do not match.
          </p>
        )}
        {setup.isError && (
          <p role="alert" className="text-sm text-bad">
            {errorMessage(setup.error)}
          </p>
        )}
        <Button
          type="submit"
          variant="primary"
          className="w-full"
          loading={setup.isPending}
          disabled={mismatch || !username.trim() || !password}
        >
          Create account
        </Button>
      </form>
    </AuthLayout>
  )
}
