import { useId } from 'react'
import * as RSwitch from '@radix-ui/react-switch'
import styles from './Switch.module.css'

export function Switch({ checked, onCheckedChange, label }: { checked: boolean; onCheckedChange: (c: boolean) => void; label: string }) {
  const id = useId()
  return (
    <span className={styles.row}>
      <RSwitch.Root id={id} className={styles.switch} checked={checked} onCheckedChange={onCheckedChange}>
        <RSwitch.Thumb className={styles.thumb} />
      </RSwitch.Root>
      <label htmlFor={id}>{label}</label>
    </span>
  )
}
