import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useHistoryCalendar, type HistoryFilters } from '@/api'
import { useFormat } from '@/lib/format'
import { AppLink, Button, Card, ErrorState, Skeleton } from '@/ui'
import { entityPath, familyOf } from '@/lib/routes'
import { shiftMonth } from './links'
import s from './history.module.css'

const MAX_ITEMS = 3

/** Month calendar of the history (?month=YYYY-MM): views per day and the titles watched. */
export function HistoryCalendar({ month, today, filters, onMonth }: { month: string; today: string; filters: HistoryFilters; onMonth: (m: string) => void }) {
  const { t } = useTranslation()
  const f = useFormat()
  const q = useHistoryCalendar(month, filters)
  const days = Array.isArray(q.data) ? q.data : undefined
  const [y, m] = month.split('-').map(Number)
  const offset = (new Date(y, m - 1, 1).getDay() + 6) % 7
  const label = new Intl.DateTimeFormat(f.locale, { month: 'long', year: 'numeric' }).format(new Date(y, m - 1, 1))
  const weekdays = f.weekdays()

  return (
    <Card>
      <div className={s.calHead}>
        <h2 aria-live="polite">{label}</h2>
        <div className={s.calNav}>
          <Button size="sm" variant="ghost" iconOnly aria-label={t('app.history.calendar.prev')} icon={<ChevronLeft size={16} />} onClick={() => onMonth(shiftMonth(month, -1))} />
          <Button size="sm" variant="ghost" onClick={() => onMonth(today.slice(0, 7))} disabled={month === today.slice(0, 7)}>{t('app.history.calendar.current')}</Button>
          <Button size="sm" variant="ghost" iconOnly aria-label={t('app.history.calendar.next')} icon={<ChevronRight size={16} />} onClick={() => onMonth(shiftMonth(month, 1))} />
        </div>
      </div>
      {q.isError && !days ? (
        <ErrorState title={t('app.history.error')} onRetry={() => q.refetch()} />
      ) : (
        <div className={s.calGrid} role="list" aria-label={label}>
          {weekdays.map((d) => <div key={d} className={s.calDow} aria-hidden="true">{d}</div>)}
          {Array.from({ length: offset }, (_, i) => <div key={`b${i}`} className={`${s.calDay} ${s.calBlank}`} aria-hidden="true" />)}
          {!days
            ? Array.from({ length: 28 }, (_, i) => <div key={i} className={s.calDay}><Skeleton width="40%" /><Skeleton width="80%" /></div>)
            : days.map((d) => {
              const items = d.items ?? []
              return (
                <div key={d.date} role="listitem" className={[s.calDay, d.date === today && s.calToday].filter(Boolean).join(' ')} aria-label={`${d.date} · ${t('app.history.calendar.views', { count: d.views })}`}>
                  <div className={s.calTop}>
                    <span className={s.calNum}>{Number(d.date.slice(8))}</span>
                    {d.views > 0 && <span className={s.calCount}>{t('app.history.calendar.views', { count: d.views })}</span>}
                  </div>
                  {items.slice(0, MAX_ITEMS).map((it) => (
                    <div key={it.id} className={s.calItem} title={`${it.name} · ${it.user}`}>
                      <i style={{ background: `var(--t-${familyOf(it.kind)})` }} aria-hidden="true" />
                      <AppLink to={entityPath(it.kind, it.id)}>{it.name}</AppLink>
                    </div>
                  ))}
                  {items.length > MAX_ITEMS && <span className={s.calMore}>{t('app.history.calendar.more', { count: items.length - MAX_ITEMS })}</span>}
                </div>
              )
            })}
        </div>
      )}
    </Card>
  )
}
