import styles from './Meter.module.css'

export function Meter({ value, tone = 'accent', thin, label }: { value: number; tone?: 'accent' | 'serious' | 'muted' | 'series'; thin?: boolean; label: string }) {
  const pct = Math.round(Math.min(1, Math.max(0, value)) * 100)
  return (
    <div className={[styles.meter, thin && styles.thin].filter(Boolean).join(' ')} role="progressbar" aria-label={label} aria-valuemin={0} aria-valuemax={100} aria-valuenow={pct}>
      <span className={styles[tone]} style={{ width: `${pct}%` }} />
    </div>
  )
}
