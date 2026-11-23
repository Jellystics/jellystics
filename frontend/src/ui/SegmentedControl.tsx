import type { ReactNode } from 'react'
import * as ToggleGroup from '@radix-ui/react-toggle-group'
import styles from './SegmentedControl.module.css'

export function SegmentedControl<T extends string>({ value, onValueChange, options, 'aria-label': ariaLabel }: {
  value: T
  onValueChange: (v: T) => void
  options: { value: T; label: ReactNode; ariaLabel?: string }[]
  'aria-label': string
}) {
  return (
    <ToggleGroup.Root type="single" className={styles.seg} value={value} aria-label={ariaLabel} onValueChange={(v) => { if (v) onValueChange(v as T) }}>
      {options.map((o) => (
        <ToggleGroup.Item key={o.value} value={o.value} className={styles.item} aria-label={o.ariaLabel}>{o.label}</ToggleGroup.Item>
      ))}
    </ToggleGroup.Root>
  )
}
