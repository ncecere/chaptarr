import { useState, type ReactNode } from 'react'
import { Button } from './Button'
import { Checkbox } from './Checkbox'
import { Modal } from './Modal'

interface ConfirmDialogProps {
  open: boolean
  title: string
  message: ReactNode
  confirmLabel?: string
  danger?: boolean
  /** Optional extra choice, passed back to onConfirm. */
  checkbox?: { label: string; defaultChecked?: boolean }
  pending?: boolean
  onConfirm: (checked: boolean) => void
  onCancel: () => void
}

export function ConfirmDialog(props: ConfirmDialogProps) {
  // The inner component mounts on open, so the checkbox resets every time.
  return props.open ? <ConfirmInner {...props} /> : null
}

function ConfirmInner({
  open,
  title,
  message,
  confirmLabel = 'Confirm',
  danger,
  checkbox,
  pending,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const [checked, setChecked] = useState(checkbox?.defaultChecked ?? false)
  return (
    <Modal
      open={open}
      onClose={onCancel}
      title={title}
      size="sm"
      busy={pending}
      footer={
        <>
          <Button onClick={onCancel} disabled={pending}>
            Cancel
          </Button>
          <Button
            variant={danger ? 'danger' : 'primary'}
            loading={pending}
            onClick={() => onConfirm(checked)}
          >
            {confirmLabel}
          </Button>
        </>
      }
    >
      <div className="space-y-4 text-sm text-muted">
        <div>{message}</div>
        {checkbox && (
          <Checkbox checked={checked} onChange={setChecked} label={checkbox.label} className="text-fg" />
        )}
      </div>
    </Modal>
  )
}
