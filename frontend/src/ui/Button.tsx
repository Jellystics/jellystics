import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react'
import styles from './Button.module.css'

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
export type ButtonSize = 'sm' | 'md' | 'lg'

export function buttonClass({ variant = 'secondary', size = 'md', iconOnly, className }: { variant?: ButtonVariant; size?: ButtonSize; iconOnly?: boolean; className?: string } = {}): string {
  return [styles.btn, styles[variant], size !== 'md' && styles[size], iconOnly && styles.iconOnly, className].filter(Boolean).join(' ')
}

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant
  size?: ButtonSize
  icon?: ReactNode
  /** Requires aria-label. */
  iconOnly?: boolean
  asChild?: false
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = 'secondary', size = 'md', icon, iconOnly, asChild: _asChild, className, children, type = 'button', ...rest },
  ref,
) {
  const cls = buttonClass({ variant, size, iconOnly, className })
  return (
    <button ref={ref} type={type} className={cls} {...rest}>
      {icon}
      {iconOnly ? null : children}
    </button>
  )
})
