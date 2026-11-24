import { useTranslation } from 'react-i18next'
import { formatPercent } from '@/lib/format'
import styles from './SegBar.module.css'

export type SegPart = { label: string; value: number; color: string; ink?: string; display?: string }

export function SegBar({ parts, thin }: { parts: SegPart[]; thin?: boolean }) {
  const { i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'fr'
  const total = parts.reduce((a, p) => a + p.value, 0) || 1
  // An empty part would still show its padding (and a gap) as a sliver of color: it only goes in the legend.
  const drawn = parts.filter((p) => p.value > 0)
  if (thin) {
    return (
      <div className={styles.thin} role="list">
        {drawn.map((p) => (
          <span key={p.label} role="listitem" tabIndex={0} aria-label={`${p.label} : ${p.display ?? formatPercent(p.value / total, locale)}`} style={{ flex: p.value, background: p.color }} />
        ))}
      </div>
    )
  }
  return (
    <>
      <div className={styles.tall} role="list">
        {drawn.map((p) => {
          const share = p.value / total
          const pct = formatPercent(share, locale)
          return (
            <span key={p.label} role="listitem" tabIndex={0} aria-label={`${p.label} : ${p.display ?? pct}`} style={{ flex: p.value, background: p.color, color: p.ink ?? 'var(--ink-on-accent)' }}>
              {share > 0.09 ? pct : ''}
            </span>
          )
        })}
      </div>
      <div className={styles.legend}>
        {parts.map((p) => (
          <span key={p.label}><i style={{ background: p.color }} />{p.label}</span>
        ))}
      </div>
    </>
  )
}
