import { useState } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import * as RPopover from '@radix-ui/react-popover'
import { CalendarDays, Check, ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { addDays, previousPeriod, type Period, type Preset } from '@/lib/period'
import { usePeriod } from '@/lib/useUrlState'
import { formatDay } from '@/lib/format'
import { Button } from './Button'
import { Input } from './Input'
import menu from './Menu.module.css'
import styles from './PeriodPicker.module.css'

const PRESETS: Preset[] = ['today', '7d', '30d', '90d', 'mtd', 'ytd', 'all']
const toLocalDate = (s: string) => { const [y, m, d] = s.split('-').map(Number); return new Date(y, m - 1, d) }

export function PeriodPicker() {
  const { t, i18n } = useTranslation()
  const [period, update] = usePeriod()
  const [customOpen, setCustomOpen] = useState(false)
  const locale = i18n.resolvedLanguage || 'fr'
  const fmt = (s: string) => formatDay(toLocalDate(s), locale)
  const range = (from: string, to: string) => `${fmt(from)} – ${fmt(addDays(to, -1))}`

  const label = period.preset === 'custom' ? range(period.from, period.to) : t(`app.period.${period.preset}`)
  const prev = period.compare && period.preset !== 'all' ? previousPeriod(period) : null

  return (
    <RPopover.Root open={customOpen} onOpenChange={setCustomOpen}>
      <DropdownMenu.Root>
        <RPopover.Anchor asChild>
          <span>
            <DropdownMenu.Trigger asChild>
              <Button variant="secondary" icon={<CalendarDays size={14} strokeWidth={1.6} aria-hidden />}>
                {label}
                {prev && <span className={styles.vs}>{t('app.period.vs', { range: range(prev.from, prev.to) })}</span>}
                <ChevronDown size={14} strokeWidth={1.6} aria-hidden />
              </Button>
            </DropdownMenu.Trigger>
          </span>
        </RPopover.Anchor>
        <DropdownMenu.Portal>
          <DropdownMenu.Content className={menu.pop} align="end" sideOffset={4} onCloseAutoFocus={(e) => { if (customOpen) e.preventDefault() }}>
            <DropdownMenu.RadioGroup value={period.preset} onValueChange={(v) => update({ preset: v as Period['preset'] })}>
              {PRESETS.map((p) => (
                <DropdownMenu.RadioItem key={p} value={p} className={menu.item}>
                  <span className={menu.check}><DropdownMenu.ItemIndicator><Check size={12} strokeWidth={2.5} aria-hidden /></DropdownMenu.ItemIndicator></span>
                  {t(`app.period.${p}`)}
                </DropdownMenu.RadioItem>
              ))}
            </DropdownMenu.RadioGroup>
            <DropdownMenu.Separator className={menu.sep} />
            <DropdownMenu.Item className={menu.item} onSelect={() => setCustomOpen(true)}>
              <span className={menu.check} />
              {t('app.period.custom')}
            </DropdownMenu.Item>
            <DropdownMenu.Separator className={menu.sep} />
            <DropdownMenu.CheckboxItem className={menu.item} checked={period.compare && period.preset !== 'all'} disabled={period.preset === 'all'} onCheckedChange={(c) => update({ compare: c === true })}>
              <span className={menu.check}><DropdownMenu.ItemIndicator><Check size={12} strokeWidth={2.5} aria-hidden /></DropdownMenu.ItemIndicator></span>
              {t('app.period.compare')}
            </DropdownMenu.CheckboxItem>
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
      <RPopover.Portal>
        <RPopover.Content className={menu.pop} align="end" sideOffset={4}>
          <CustomRange
            from={period.from}
            to={addDays(period.to, -1)}
            onApply={(from, to) => { update({ preset: 'custom', from, to: addDays(to, 1) }); setCustomOpen(false) }}
          />
        </RPopover.Content>
      </RPopover.Portal>
    </RPopover.Root>
  )
}

function CustomRange({ from: f0, to: t0, onApply }: { from: string; to: string; onApply: (from: string, to: string) => void }) {
  const { t } = useTranslation()
  const [from, setFrom] = useState(f0)
  const [to, setTo] = useState(t0)
  const invalid = !from || !to || from > to
  return (
    <form className={styles.custom} onSubmit={(e) => { e.preventDefault(); if (!invalid) onApply(from, to) }}>
      <label className={styles.row}>{t('app.period.from')}<Input type="date" value={from} max={to || undefined} onChange={(e) => setFrom(e.target.value)} /></label>
      <label className={styles.row}>{t('app.period.to')}<Input type="date" value={to} min={from || undefined} onChange={(e) => setTo(e.target.value)} /></label>
      {invalid && <span className={styles.error} role="alert">{t('app.period.invalid')}</span>}
      <div className={styles.actions}><Button type="submit" variant="primary" disabled={invalid}>{t('app.period.apply')}</Button></div>
    </form>
  )
}
