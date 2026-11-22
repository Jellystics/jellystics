import { useId } from 'react'
import * as RCheckbox from '@radix-ui/react-checkbox'
import { Check } from 'lucide-react'
import styles from './Switch.module.css'

export function Checkbox({ checked, onCheckedChange, label }: { checked: boolean; onCheckedChange: (c: boolean) => void; label: string }) {
  const id = useId()
  return (
    <span className={styles.row}>
      <RCheckbox.Root id={id} className={styles.checkbox} checked={checked} onCheckedChange={(c) => onCheckedChange(c === true)}>
        <RCheckbox.Indicator><Check size={12} strokeWidth={2.5} aria-hidden /></RCheckbox.Indicator>
      </RCheckbox.Root>
      <label htmlFor={id}>{label}</label>
    </span>
  )
}
