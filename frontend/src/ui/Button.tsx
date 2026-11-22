import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react'
import styles from './Button.module.css'

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  icon?: ReactNode
  /** Requires aria-label. */
  iconOnly?: boolean
  asChild?: false
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = 'secondary', size = 'md', icon, iconOnly, asChild: _asChild, className, children, type = 'button', ...rest },
  ref,
) {
  const cls = [styles.btn, styles[variant], size !== 'md' && styles[size], iconOnly && styles.iconOnly, className].filter(Boolean).join(' ')
  return (
    <button ref={ref} type={type} className={cls} {...rest}>
      {icon}
      {iconOnly ? null : children}
    </button>
  )
})
