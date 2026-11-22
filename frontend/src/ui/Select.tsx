import * as RSelect from '@radix-ui/react-select'
import { Check, ChevronDown } from 'lucide-react'
import styles from './Select.module.css'
import menu from './Menu.module.css'

export function Select({ value, onValueChange, options, 'aria-label': ariaLabel }: {
  value: string
  onValueChange: (v: string) => void
  options: { value: string; label: string }[]
  'aria-label': string
}) {
  return (
    <RSelect.Root value={value} onValueChange={onValueChange}>
      <RSelect.Trigger className={styles.select} aria-label={ariaLabel}>
        <RSelect.Value />
        <RSelect.Icon><ChevronDown size={14} strokeWidth={1.6} aria-hidden /></RSelect.Icon>
      </RSelect.Trigger>
      <RSelect.Portal>
        <RSelect.Content className={menu.pop} position="popper" sideOffset={4}>
          <RSelect.Viewport>
            {options.map((o) => (
              <RSelect.Item key={o.value} value={o.value} className={menu.item}>
                <span className={menu.check}><RSelect.ItemIndicator><Check size={12} strokeWidth={2.5} aria-hidden /></RSelect.ItemIndicator></span>
                <RSelect.ItemText>{o.label}</RSelect.ItemText>
              </RSelect.Item>
            ))}
          </RSelect.Viewport>
        </RSelect.Content>
      </RSelect.Portal>
    </RSelect.Root>
  )
}
