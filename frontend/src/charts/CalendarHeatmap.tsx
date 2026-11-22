import { useTranslation } from 'react-i18next'
import { useFormat } from '@/lib/format'
import { useChartTooltip } from './ChartTooltip'
import { useChartWidth } from './useChartWidth'
import { SEQ, seqIndex, weekdayLong } from './labels'
import styles from './charts.module.css'

const DAY_MS = 86400000
const parse = (s: string) => { const [y, m, d] = s.split('-').map(Number); return new Date(y, m - 1, d) }
const utc = (d: Date) => Date.UTC(d.getFullYear(), d.getMonth(), d.getDate())

export function CalendarHeatmap({ days }: { days: { date: string; seconds: number }[] }) {
  const { t } = useTranslation()
  const f = useFormat()
  const [ref, width] = useChartWidth()
  const tip = useChartTooltip()
  const m = { t: 16, l: 28, b: 4 }, gap = 2
  const sorted = days.map((d) => ({ ...d, day: parse(d.date) })).sort((a, b) => a.day.getTime() - b.day.getTime())
  const first = sorted[0]?.day
  const offset = first ? (first.getDay() + 6) % 7 : 0 // Monday = 0
  const cells = sorted.map((d) => ({ ...d, idx: Math.round((utc(d.day) - utc(first)) / DAY_MS) + offset }))
  const weeks = cells.length ? Math.floor(cells[cells.length - 1].idx / 7) + 1 : 1
  const cw = (width - m.l) / weeks
  const height = m.t + cw * 7 + m.b
  const max = cells.reduce((a, c) => Math.max(a, c.seconds), 0)
  const short = f.weekdays(), full = weekdayLong(f.locale)
  const monthFmt = new Intl.DateTimeFormat(f.locale, { month: 'short' })
  const monthLabels = first
    ? Array.from({ length: weeks }, (_, w) => ({ w, monday: new Date(first.getFullYear(), first.getMonth(), first.getDate() - offset + 7 * w) })).filter(({ monday }) => monday.getDate() <= 7)
    : []

  return (
    <div className={styles.chart} ref={ref}>
      <svg width={width} height={height} role="img" aria-label={t('app.chart.calendar')}>
        {monthLabels.map(({ w, monday }) => (
          <text key={w} x={m.l + w * cw} y={m.t - 5}>{monthFmt.format(monday)}</text>
        ))}
        {[0, 2, 4].map((d) => (
          <text key={d} x={m.l - 6} y={m.t + d * cw + cw - 4} textAnchor="end">{short[d]}</text>
        ))}
        {cells.map((c) => {
          const w = Math.floor(c.idx / 7), d = c.idx % 7
          return (
            <rect
              key={c.date} x={m.l + w * cw} y={m.t + d * cw} width={Math.max(1, cw - gap)} height={Math.max(1, cw - gap)} rx={2}
              fill={SEQ[seqIndex(c.seconds, max, 6)]}
              {...tip.hit(`${full[d]} ${f.day(c.day)}`, [{ label: t('app.chart.playTime'), value: c.seconds ? f.duration(c.seconds) : '—' }])}
            />
          )
        })}
      </svg>
      {tip.node}
    </div>
  )
}
