import { useState, type FormEvent } from 'react'
import { Button } from '../../components/Button'
import { Field, Input } from '../../components/Form'
import { useLogin } from '../../lib/auth'
import { errorMessage } from '../../lib/queryClient'
import { useDocumentTitle } from '../../lib/hooks'
import { AuthLayout } from './AuthLayout'

export function LoginPage() {
  useDocumentTitle('Sign in')
  const login = useLogin()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const submit = (e: FormEvent) => {
    e.preventDefault()
    login.mutate({ username: username.trim(), password })
  }

  return (
    <AuthLayout title="Sign in" subtitle="Welcome back.">
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
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </Field>
        {login.isError && (
          <p role="alert" className="text-sm text-bad">
            {errorMessage(login.error)}
          </p>
        )}
        <Button type="submit" variant="primary" className="w-full" loading={login.isPending}>
          Sign in
        </Button>
      </form>
    </AuthLayout>
  )
}
