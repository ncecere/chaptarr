import { Copy, Eye, EyeOff, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { Button, IconButton } from '../../components/Button'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { Field, Input } from '../../components/Form'
import { QueryView } from '../../components/QueryView'
import { FormSection } from '../../components/FormLayout'
import { useUsername } from '../../lib/auth'
import { useChangePassword, useRegenerateApiKey } from '../../lib/mutations'
import { useStatus } from '../../lib/queries'
import { toast } from '../../lib/toast'
import { copyText } from '../../lib/text'

function PasswordForm() {
  const username = useUsername()
  const change = useChangePassword()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const mismatch = confirm !== '' && next !== confirm

  return (
    <FormSection title="Password" description={`Signed in as ${username}.`}>
      <form
        className="max-w-sm space-y-4"
        onSubmit={(e) => {
          e.preventDefault()
          if (next !== confirm) return
          change.mutate(
            { currentPassword: current, newPassword: next },
            {
              onSuccess: () => {
                setCurrent('')
                setNext('')
                setConfirm('')
              },
            },
          )
        }}
      >
        <input type="text" value={username} autoComplete="username" readOnly hidden />
        <Field label="Current password">
          <Input
            type="password"
            autoComplete="current-password"
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
            required
          />
        </Field>
        <Field label="New password">
          <Input
            type="password"
            autoComplete="new-password"
            value={next}
            onChange={(e) => setNext(e.target.value)}
            required
          />
        </Field>
        <Field label="Confirm new password">
          <Input
            type="password"
            autoComplete="new-password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            aria-invalid={mismatch}
            required
          />
        </Field>
        {mismatch && <p className="text-sm text-bad">Passwords do not match.</p>}
        <Button
          type="submit"
          variant="primary"
          loading={change.isPending}
          disabled={!current || !next || next !== confirm}
        >
          Change password
        </Button>
      </form>
    </FormSection>
  )
}

function ApiKeySection() {
  const query = useStatus()
  const regenerate = useRegenerateApiKey()
  const [revealed, setRevealed] = useState(false)
  const [confirming, setConfirming] = useState(false)

  return (
    <FormSection
      title="API key"
      description="Send this as the X-Api-Key header to use the Chaptarr API from scripts and other tools."
    >
      <QueryView query={query}>
        {(status) => (
          <div>
            <div className="flex gap-2">
              <Input
                readOnly
                type={revealed ? 'text' : 'password'}
                value={status.apiKey}
                aria-label="API key"
                autoComplete="off"
                className="font-mono"
                onFocus={(e) => e.currentTarget.select()}
              />
              <IconButton
                label={revealed ? 'Hide API key' : 'Show API key'}
                variant="secondary"
                onClick={() => setRevealed(!revealed)}
              >
                {revealed ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
              </IconButton>
              <IconButton
                label="Copy API key"
                variant="secondary"
                onClick={async () =>
                  (await copyText(status.apiKey))
                    ? toast.success('API key copied')
                    : toast.error('Could not copy — reveal the key and copy it manually')
                }
              >
                <Copy className="size-4" />
              </IconButton>
            </div>
            <Button
              className="mt-3"
              icon={<RefreshCw className="size-4" />}
              onClick={() => setConfirming(true)}
            >
              Regenerate
            </Button>
          </div>
        )}
      </QueryView>

      <ConfirmDialog
        open={confirming}
        title="Regenerate API key?"
        danger
        confirmLabel="Regenerate"
        message="The current key stops working immediately. Anything using it will need the new key."
        pending={regenerate.isPending}
        onCancel={() => setConfirming(false)}
        onConfirm={() => regenerate.mutate(undefined, { onSettled: () => setConfirming(false) })}
      />
    </FormSection>
  )
}

export function AccountSettings() {
  return (
    // One parent, so the first section drops its top rule like the other settings pages.
    <div>
      <PasswordForm />
      <ApiKeySection />
    </div>
  )
}
