import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { cn } from '../lib/cn'
import { Spinner } from './Spinner'

/**
 * primary     the one main action of a view
 * secondary   everything else that needs a visible button
 * ghost       low-emphasis actions and icon buttons
 * danger      destructive actions inline: quiet until hovered
 * danger-solid  the confirming button of a destructive dialog
 */
type Variant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'danger-solid'
type Size = 'sm' | 'md'

const base =
  'inline-flex shrink-0 items-center justify-center whitespace-nowrap rounded-control font-medium select-none ' +
  'transition-[color,background-color,border-color,box-shadow,transform] duration-150 active:scale-[0.98] ' +
  'disabled:cursor-not-allowed disabled:opacity-50 disabled:active:scale-100'

const variants: Record<Variant, string> = {
  primary: 'bg-accent text-accent-fg shadow-raised hover:bg-accent-hover',
  secondary:
    'border border-line-strong bg-surface-2 text-fg shadow-raised hover:border-faint/60 hover:bg-surface-3',
  ghost: 'text-muted hover:bg-surface-2 hover:text-fg',
  danger: 'border border-transparent text-bad hover:border-bad/25 hover:bg-bad/10',
  'danger-solid': 'bg-bad text-on-status shadow-raised hover:bg-bad/85',
}

const sizes: Record<Size, string> = {
  sm: 'h-7 px-2.5 text-xs gap-1.5',
  md: 'h-8 px-3 text-sm gap-1.5',
}

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  loading?: boolean
  icon?: ReactNode
}

export function Button({
  variant = 'secondary',
  size = 'md',
  loading,
  icon,
  disabled,
  className,
  children,
  type = 'button',
  ...rest
}: ButtonProps) {
  return (
    <button
      type={type}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cn(base, variants[variant], sizes[size], className)}
      {...rest}
    >
      {loading ? <Spinner className="size-3.5" /> : icon}
      {children}
    </button>
  )
}

interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'aria-label'> {
  label: string
  size?: Size
  variant?: Variant
  loading?: boolean
}

/** Icon-only button; `label` becomes the accessible name and tooltip. */
export function IconButton({
  label,
  size = 'md',
  variant = 'ghost',
  loading,
  disabled,
  className,
  children,
  type = 'button',
  ...rest
}: IconButtonProps) {
  return (
    <button
      type={type}
      aria-label={label}
      title={label}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cn(base, variants[variant], size === 'sm' ? 'size-7' : 'size-8', className)}
      {...rest}
    >
      {loading ? <Spinner className="size-4" /> : children}
    </button>
  )
}
