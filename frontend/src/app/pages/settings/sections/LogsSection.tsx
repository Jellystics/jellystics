import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { RotateCw } from 'lucide-react'
import { Badge, Button, Card, CardBody, CardHeader, DataTable, EmptyState, ErrorState, SearchInput, SegmentedControl, type BadgeTone } from '@/ui'
import { apiFetch } from '@/lib/api'
import { asArray } from '../net'
import styles from '../SettingsPage.module.css'

type Level = 'info' | 'warn' | 'error'
interface LogRow { id: number; level: Level; message: string; time: string; task: string }

const TONE: Record<Level, BadgeTone> = { info: 'neutral', warn: 'warning', error: 'critical' }

function toLogRows(raw: unknown): LogRow[] {
  return asArray<Record<string, unknown>>(raw).map((r, i) => {
    const result = String(r.Result ?? '').toLowerCase()
    const text = String(r.Log ?? r.Result ?? '')
    let message = text
    try {
      const parsed = JSON.parse(text)
      if (Array.isArray(parsed)) message = parsed.map((e: { Message?: string }) => e.Message ?? '').filter(Boolean).join(' | ')
    } catch { /* plain text */ }
    return { id: i, level: result === 'failed' ? 'error' : result === 'running' ? 'warn' : 'info', message, time: String(r.TimeRun ?? ''), task: String(r.Name ?? '') }
  })
}

export function LogsSection() {
  const { t, i18n } = useTranslation()
  const q = useQuery({ queryKey: ['settings', 'logs'], queryFn: () => apiFetch<unknown>('/logs/getLogs') })
  const [level, setLevel] = useState<'all' | Level>('all')
  const [search, setSearch] = useState('')
  const rows = useMemo(() => toLogRows(q.data), [q.data])
  const shown = useMemo(() => {
    const s = search.trim().toLowerCase()
    return rows.filter((r) => (level === 'all' || r.level === level) && (!s || `${r.task} ${r.message}`.toLowerCase().includes(s)))
  }, [rows, level, search])
  const fmt = new Intl.DateTimeFormat(i18n.resolvedLanguage || 'fr', { dateStyle: 'medium', timeStyle: 'medium' })
  const date = (s: string) => { const d = new Date(s); return Number.isNaN(d.getTime()) ? s : fmt.format(d) }

  const columns: ColumnDef<LogRow>[] = [
    { id: 'level', header: t('app.settings.logs.level'), cell: ({ row }) => <Badge tone={TONE[row.original.level]}>{t(`app.settings.logs.levels.${row.original.level}`)}</Badge> },
    { id: 'time', header: t('app.settings.logs.date'), cell: ({ row }) => date(row.original.time) },
    { id: 'task', header: t('app.settings.logs.task'), cell: ({ row }) => row.original.task || '—' },
    { id: 'message', header: t('app.settings.logs.message'), cell: ({ row }) => <span className={styles.pre}>{row.original.message}</span> },
  ]

  return (
    <Card>
      <CardHeader
        title={t('app.settings.logs.title')}
        action={<Button size="sm" icon={<RotateCw size={16} strokeWidth={1.6} />} onClick={() => q.refetch()}>{t('app.settings.logs.refresh')}</Button>}
      />
      <CardBody>
        <div className={styles.row} style={{ marginBottom: 'var(--s-3)' }}>
          <div className={styles.grow}><SearchInput value={search} onChange={setSearch} placeholder={t('app.settings.logs.search')} /></div>
          <SegmentedControl
            aria-label={t('app.settings.logs.level')} value={level} onValueChange={(v) => setLevel(v as 'all' | Level)}
            options={[{ value: 'all', label: t('app.settings.logs.all') }, { value: 'info', label: t('app.settings.logs.levels.info') }, { value: 'warn', label: t('app.settings.logs.levels.warn') }, { value: 'error', label: t('app.settings.logs.levels.error') }]}
          />
        </div>
        {q.isError ? <ErrorState title={t('app.error.generic')} onRetry={() => q.refetch()} /> : (
          <DataTable columns={columns} data={shown} loading={q.isLoading} dense empty={<EmptyState title={t('app.settings.logs.empty')} />} />
        )}
      </CardBody>
    </Card>
  )
}
