import { useId, useMemo, useState, type ReactNode } from 'react'
import * as RPopover from '@radix-ui/react-popover'
import { Check } from 'lucide-react'
import { Avatar } from './Avatar'
import { SearchInput } from './SearchInput'
import menu from './Menu.module.css'
import styles from './SearchMenu.module.css'

export type SearchMenuOption = { value: string; label: string; /** Show the user's avatar in front of the label. */ avatarId?: string }

const norm = (s: string) => s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()

/**
 * Fixed-size picker for long lists (users, clients, devices): a popover with a
 * search field and a scrolling listbox, keyboard driven (↑ ↓ Enter, Esc).
 */
export function SearchMenu({ trigger, options, value, onSelect, searchPlaceholder, emptyLabel, align = 'start' }: {
  trigger: ReactNode
  options: SearchMenuOption[]
  value?: string
  onSelect: (value: string) => void
  searchPlaceholder: string
  emptyLabel: string
  align?: 'start' | 'end'
}) {
  const [open, setOpen] = useState(false)
  const [q, setQ] = useState('')
  const [active, setActive] = useState(0)
  const listId = useId()
  const shown = useMemo(() => { const n = norm(q.trim()); return n ? options.filter((o) => norm(o.label).includes(n)) : options }, [options, q])
  const idx = Math.min(active, Math.max(shown.length - 1, 0))

  const choose = (v: string) => { onSelect(v); setOpen(false) }
  const onOpenChange = (o: boolean) => { setOpen(o); if (o) { setQ(''); setActive(Math.max(0, options.findIndex((x) => x.value === value))) } }
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); setActive(Math.min(idx + 1, shown.length - 1)) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setActive(Math.max(idx - 1, 0)) }
    else if (e.key === 'Enter' && shown[idx]) { e.preventDefault(); choose(shown[idx].value) }
  }
  const optId = (i: number) => `${listId}-${i}`

  return (
    <RPopover.Root open={open} onOpenChange={onOpenChange}>
      <RPopover.Trigger asChild>{trigger}</RPopover.Trigger>
      <RPopover.Portal>
        <RPopover.Content className={styles.pop} align={align} sideOffset={4} onKeyDown={onKeyDown}>
          <div className={styles.search}>
            <SearchInput
              value={q}
              onChange={(v) => { setQ(v); setActive(0) }}
              placeholder={searchPlaceholder}
              inputProps={{ role: 'combobox', 'aria-expanded': true, 'aria-controls': listId, 'aria-activedescendant': shown[idx] ? optId(idx) : undefined, autoFocus: true }}
            />
          </div>
          <ul id={listId} role="listbox" className={styles.list} aria-label={searchPlaceholder}>
            {shown.length === 0 && <li className={styles.empty}>{emptyLabel}</li>}
            {shown.map((o, i) => (
              <li
                key={o.value}
                id={optId(i)}
                role="option"
                aria-selected={o.value === value}
                data-highlighted={i === idx ? '' : undefined}
                className={menu.item}
                onMouseEnter={() => setActive(i)}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => choose(o.value)}
              >
                {value !== undefined && <span className={menu.check}>{o.value === value && <Check size={12} strokeWidth={2.5} aria-hidden />}</span>}
                {o.avatarId && <Avatar id={o.avatarId} name={o.label} size="sm" />}
                <span className={styles.label}>{o.label}</span>
              </li>
            ))}
          </ul>
        </RPopover.Content>
      </RPopover.Portal>
    </RPopover.Root>
  )
}
