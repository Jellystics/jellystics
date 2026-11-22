import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Button, Card, CardBody, CardHeader, SegmentedControl } from '@/ui'
import styles from './charts.module.css'

type Legend = { label: string; color: string; line?: boolean }[]
type TableData = { columns: string[]; rows: (string | number)[][] }

const csvCell = (v: string | number) => { const s = String(v); return /[",\r\n;]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s }
export const toCsv = (t: TableData) => [t.columns, ...t.rows].map((r) => r.map(csvCell).join(',')).join('\n')

function download(name: string, table: TableData) {
  const url = URL.createObjectURL(new Blob(['﻿' + toCsv(table)], { type: 'text/csv;charset=utf-8' }))
  const a = document.createElement('a')
  a.href = url
  a.download = `${name}.csv`
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

export function ChartFrame({ title, subtitle, legend, table, csvName, action, children }: {
  title: string; subtitle?: string; legend?: Legend; table: TableData; csvName: string; action?: ReactNode; children: ReactNode
}) {
  const { t } = useTranslation()
  const [view, setView] = useState<'chart' | 'table'>('chart')
  return (
    <Card>
      <CardHeader
        title={title}
        subtitle={subtitle}
        action={
          <div className={styles.actions}>
            {action}
            <SegmentedControl
              aria-label={title}
              value={view}
              onValueChange={setView}
              options={[
                { value: 'chart', label: t('app.chart.chart') },
                { value: 'table', label: t('app.chart.table') },
              ]}
            />
            <Button size="sm" variant="ghost" onClick={() => download(csvName, table)}>{t('app.chart.csv')}</Button>
          </div>
        }
      />
      <CardBody>
        {legend && legend.length > 0 && (
          <div className={styles.legend}>
            {legend.map((l) => <span key={l.label}><i className={l.line ? styles.line : undefined} style={{ background: l.color }} />{l.label}</span>)}
          </div>
        )}
        {view === 'chart' ? children : (
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead><tr>{table.columns.map((c) => <th key={c} scope="col">{c}</th>)}</tr></thead>
              <tbody>{table.rows.map((r, i) => <tr key={i}>{r.map((c, j) => <td key={j}>{c}</td>)}</tr>)}</tbody>
            </table>
          </div>
        )}
      </CardBody>
    </Card>
  )
}
