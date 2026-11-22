import { useMemo, useState, type KeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { MoonStar, RefreshCw, type LucideIcon } from 'lucide-react'
import { Dialog, SearchInput, toast } from '@/ui'
import { apiFetch } from '@/lib/api'
import { useThemeMode } from '@/lib/themeMode'
import { PERIOD_PARAMS } from '@/lib/period'
import { NAV_ENTRIES } from './nav'
import styles from './CommandPalette.module.css'

type Command = { id: string; label: string; icon: LucideIcon; run: () => void }

export function CommandPalette({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { search } = useLocation()
  const { toggle } = useThemeMode()
  const [query, setQuery] = useState('')
  const [active, setActive] = useState(0)

  const commands = useMemo<Command[]>(() => {
    const cur = new URLSearchParams(search)
    const keep = new URLSearchParams()
    PERIOD_PARAMS.forEach((k) => { const v = cur.get(k); if (v !== null) keep.set(k, v) })
    const qs = keep.toString() ? `?${keep}` : ''
    return [
      ...NAV_ENTRIES.map((e): Command => ({ id: `nav-${e.key}`, label: t(`app.nav.${e.key}`), icon: e.icon, run: () => navigate(`${e.to}${qs}`) })),
      {
        id: 'sync', label: t('app.command.sync'), icon: RefreshCw,
        run: () => { apiFetch(`/api/runTask/${encodeURIComponent('Full Jellyfin Sync')}`, { method: 'POST' }).then(() => toast.info(t('app.command.syncStarted'))).catch(() => toast.error(t('app.command.syncFailed'))) },
      },
      { id: 'theme', label: t('app.command.toggleTheme'), icon: MoonStar, run: toggle },
    ]
  }, [t, navigate, search, toggle])

  const q = query.trim().toLowerCase()
  const shown = q ? commands.filter((c) => c.label.toLowerCase().includes(q)) : commands
  const idx = Math.min(active, Math.max(shown.length - 1, 0))

  const close = () => onOpenChange(false)
  const exec = (c: Command) => { close(); c.run() }
  const onKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); setActive((idx + 1) % Math.max(shown.length, 1)) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setActive((idx - 1 + shown.length) % Math.max(shown.length, 1)) }
    else if (e.key === 'Enter' && shown[idx]) { e.preventDefault(); exec(shown[idx]) }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) { setQuery(''); setActive(0) } onOpenChange(o) }} title={t('app.command.title')}>
      <div onKeyDown={onKeyDown} className={styles.wrap}>
        <SearchInput value={query} onChange={(v) => { setQuery(v); setActive(0) }} placeholder={t('app.command.placeholder')} />
        <ul className={styles.list} role="listbox" aria-label={t('app.command.title')}>
          {shown.map((c, i) => (
            <li key={c.id} role="option" aria-selected={i === idx}>
              <button type="button" tabIndex={-1} className={[styles.cmd, i === idx && styles.on].filter(Boolean).join(' ')} onMouseEnter={() => setActive(i)} onClick={() => exec(c)}>
                <c.icon size={16} strokeWidth={1.6} aria-hidden />{c.label}
              </button>
            </li>
          ))}
          {shown.length === 0 && <li className={styles.empty}>{t('app.command.empty')}</li>}
        </ul>
      </div>
    </Dialog>
  )
}
