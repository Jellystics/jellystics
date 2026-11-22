import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { useUsers, type MediaFamily } from '@/api'
import { Chip, Menu, SearchInput, type MenuItem } from '@/ui'
import type { HistoryUrlState } from './useHistoryFilters'
import s from './history.module.css'

const TYPES: MediaFamily[] = ['movie', 'series', 'music']
const METHODS = ['direct', 'stream', 'transcode'] as const

function AddMenu({ label, items }: { label: string; items: MenuItem[] }) {
  return (
    <Menu
      items={items}
      trigger={<button type="button" className={s.addChip}><Plus size={12} aria-hidden="true" /> {label}</button>}
    />
  )
}

/**
 * History filters bound to the URL: search (debounced), type chips, active
 * filter chips (removable) and « + » menus for user / client / device / method.
 * `clients` / `devices` are the values seen on the current page.
 */
export function FilterBar({ state, clients, devices, hideUser }: { state: HistoryUrlState; clients: string[]; devices: string[]; hideUser?: boolean }) {
  const { t } = useTranslation()
  const { values, set } = state
  const [q, setQ] = useState(values.q ?? '')
  const users = useUsers()
  const userList = Array.isArray(users.data) ? users.data : []
  const userName = userList.find((u) => u.id === values.user)?.name ?? values.user

  useEffect(() => { setQ(values.q ?? '') }, [values.q])
  useEffect(() => {
    if (q === (values.q ?? '')) return
    const id = setTimeout(() => set('q', q.trim() || undefined), 300)
    return () => clearTimeout(id)
  }, [q, values.q, set])

  return (
    <div className={s.filters} role="group" aria-label={t('app.history.filters')}>
      <span className={s.search}><SearchInput value={q} onChange={setQ} placeholder={t('app.history.search')} /></span>
      {TYPES.map((ty) => (
        <Chip key={ty} pressed={values.type === ty} onClick={() => set('type', values.type === ty ? undefined : ty)} onRemove={values.type === ty ? () => set('type') : undefined}>
          {t(`app.history.types.${ty}`)}
        </Chip>
      ))}
      {!hideUser && values.user && <Chip pressed onRemove={() => set('user')}>{t('app.history.chip.user', { value: userName })}</Chip>}
      {values.client && <Chip pressed onRemove={() => set('client')}>{t('app.history.chip.client', { value: values.client })}</Chip>}
      {values.device && <Chip pressed onRemove={() => set('device')}>{t('app.history.chip.device', { value: values.device })}</Chip>}
      {values.method && <Chip pressed onRemove={() => set('method')}>{t('app.history.chip.method', { value: t(`app.method.${values.method}`) })}</Chip>}
      <Chip pressed={values.views === '1'} onClick={() => set('views', values.views === '1' ? undefined : '1')} onRemove={values.views === '1' ? () => set('views') : undefined}>
        {t('app.history.viewsOnly')}
      </Chip>
      {!hideUser && !values.user && userList.length > 0 && (
        <AddMenu label={t('app.history.add.user')} items={userList.map((u) => ({ label: u.name, onSelect: () => set('user', u.id) }))} />
      )}
      {!values.client && clients.length > 0 && <AddMenu label={t('app.history.add.client')} items={clients.map((c) => ({ label: c, onSelect: () => set('client', c) }))} />}
      {!values.device && devices.length > 0 && <AddMenu label={t('app.history.add.device')} items={devices.map((d) => ({ label: d, onSelect: () => set('device', d) }))} />}
      {!values.method && <AddMenu label={t('app.history.add.method')} items={METHODS.map((m) => ({ label: t(`app.method.${m}`), onSelect: () => set('method', m) }))} />}
    </div>
  )
}
