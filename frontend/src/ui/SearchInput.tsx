import type { InputHTMLAttributes } from 'react'
import { Search } from 'lucide-react'
import styles from './Input.module.css'

export function SearchInput({ value, onChange, placeholder, shortcut, inputProps }: {
  value: string; onChange: (v: string) => void; placeholder: string; shortcut?: string
  /** Extra attributes for the input element (e.g. combobox ARIA wiring). */
  inputProps?: InputHTMLAttributes<HTMLInputElement>
}) {
  return (
    <span className={[styles.wrap, shortcut && styles.hasKbd].filter(Boolean).join(' ')}>
      <Search aria-hidden strokeWidth={1.6} />
      <input type="search" className={styles.input} value={value} placeholder={placeholder} aria-label={placeholder} onChange={(e) => onChange(e.target.value)} {...inputProps} />
      {shortcut && <kbd className={styles.kbd}>{shortcut}</kbd>}
    </span>
  )
}
