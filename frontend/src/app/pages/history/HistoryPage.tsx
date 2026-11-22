import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarDays, Download, List } from 'lucide-react'
import { downloadHistoryCsv, usePeriodParams } from '@/api'
import { useFormat } from '@/lib/format'
import { todayIn } from '@/lib/period'
import { useServerConfig } from '@/lib/serverConfig'
import { useSearchParam } from '@/lib/useUrlState'
import { Button, PageHeader, SegmentedControl, toast } from '@/ui'
import { FilterBar } from './FilterBar'
import { HistoryCalendar } from './HistoryCalendar'
import { HistoryTable } from './HistoryTable'
import { useHistoryFilters, useHistoryRows } from './useHistoryFilters'
import l from '../overview/layout.module.css'

const MONTH = /^\d{4}-(0[1-9]|1[0-2])$/
const uniq = (xs: string[]) => [...new Set(xs.filter(Boolean))].sort((a, b) => a.localeCompare(b))

/** /history — table (server paging/sort) or month calendar, filters in the URL. Replaces Activity + Timeline. */
export default function HistoryPage() {
  const { t } = useTranslation()
  const f = useFormat()
  const state = useHistoryFilters()
  const [viewParam, setView] = useSearchParam('view', 'table', { push: true })
  const view = viewParam === 'calendar' ? 'calendar' : 'table'
  const [monthParam, setMonth] = useSearchParam('month')
  const tz = useServerConfig().data?.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone
  const today = todayIn(tz)
  const month = monthParam && MONTH.test(monthParam) ? monthParam : today.slice(0, 7)
  const page = useHistoryRows(state)
  const rows = page.data && Array.isArray(page.data.rows) ? page.data.rows : []
  const csvParams = usePeriodParams(state.filters)
  const [exporting, setExporting] = useState(false)

  const exportCsv = async () => {
    setExporting(true)
    try { await downloadHistoryCsv(csvParams) } catch { toast.error(t('app.history.exportError')) } finally { setExporting(false) }
  }

  return (
    <div className={l.page}>
      <PageHeader
        title={t('app.history.title')}
        subtitle={view === 'table' && page.data && typeof page.data.total === 'number' ? t('app.history.subtitle', { count: page.data.total, total: f.number(page.data.total) }) : undefined}
        actions={
          <>
            <SegmentedControl
              aria-label={t('app.history.view')}
              value={view}
              onValueChange={(v) => setView(v)}
              options={[
                { value: 'table', label: <><List size={14} aria-hidden="true" /> {t('app.history.table')}</> },
                { value: 'calendar', label: <><CalendarDays size={14} aria-hidden="true" /> {t('app.history.calendar.title')}</> },
              ]}
            />
            <Button size="sm" icon={<Download size={14} aria-hidden="true" />} onClick={exportCsv} disabled={exporting}>{t('app.history.export')}</Button>
          </>
        }
      />
      <FilterBar state={state} clients={uniq(rows.map((r) => r.client))} devices={uniq(rows.map((r) => r.device))} />
      {view === 'calendar'
        ? <HistoryCalendar month={month} today={today} filters={state.filters} onMonth={(m) => setMonth(m === today.slice(0, 7) ? undefined : m)} />
        : <HistoryTable state={state} />}
    </div>
  )
}
