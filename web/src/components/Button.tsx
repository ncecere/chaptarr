import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { cn } from '../lib/cn'
import { Spinner } from './Spinner'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger'
type Size = 'sm' | 'md'

const variants: Record<Variant, string> = {
  primary: 'bg-accent text-accent-fg hover:bg-accent-hover font-medium',
  secondary: 'bg-surface-2 text-fg border border-line-strong hover:bg-surface-3',
  ghost: 'text-muted hover:text-fg hover:bg-surface-2',
  danger: 'bg-bad/15 text-bad border border-bad/30 hover:bg-bad/25',
}

const sizes: Record<Size, string> = {
  sm: 'h-7 px-2.5 text-xs gap-1.5',
  md: 'h-9 px-3.5 text-sm gap-2',
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
      className={cn(
        'inline-flex shrink-0 items-center justify-center whitespace-nowrap rounded-md transition-colors',
        'disabled:cursor-not-allowed disabled:opacity-50',
        variants[variant],
        sizes[size],
        className,
      )}
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
      className={cn(
        'inline-flex shrink-0 items-center justify-center rounded-md transition-colors',
        'disabled:cursor-not-allowed disabled:opacity-50',
        variants[variant],
        size === 'sm' ? 'size-7' : 'size-9',
        className,
      )}
      {...rest}
    >
      {loading ? <Spinner className="size-4" /> : children}
    </button>
  )
}
