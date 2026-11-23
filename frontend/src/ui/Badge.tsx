import type { ReactNode } from 'react'
import { Check, Cpu, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import styles from './Badge.module.css'

export type MediaType = 'movie' | 'series' | 'music'
export type PlayMethod = 'direct' | 'stream' | 'transcode'
export type BadgeTone = 'neutral' | 'accent' | 'good' | 'warning' | 'serious' | 'critical' | 'new'

export function Badge({ tone = 'neutral', icon, dot, children }: { tone?: BadgeTone; icon?: ReactNode; dot?: string; children?: ReactNode }) {
  return (
    <span className={[styles.badge, tone !== 'neutral' && styles[tone]].filter(Boolean).join(' ')}>
      {icon}
      {dot && <span className={styles.dot} style={{ background: dot }} aria-hidden="true" />}
      {children}
    </span>
  )
}

const METHODS = {
  direct: { tone: 'good', icon: <Check size={12} strokeWidth={1.6} />, key: 'app.method.direct' },
  stream: { tone: 'warning', icon: <Zap size={12} strokeWidth={1.6} />, key: 'app.method.stream' },
  transcode: { tone: 'serious', icon: <Cpu size={12} strokeWidth={1.6} />, key: 'app.method.transcode' },
} as const

export function MethodBadge({ method }: { method: PlayMethod }) {
  const { t } = useTranslation()
  const m = METHODS[method]
  return <Badge tone={m.tone} icon={m.icon}>{t(m.key)}</Badge>
}

export function MediaTypeBadge({ type }: { type: MediaType }) {
  const { t } = useTranslation()
  return <Badge dot={`var(--t-${type})`}>{t(`app.type.${type}`)}</Badge>
}

export function ViewBadge({ view }: { view: boolean }) {
  const { t } = useTranslation()
  return view
    ? <Badge tone="accent" dot="var(--accent)">{t('app.badge.view')}</Badge>
    : <Badge dot="var(--axis)">{t('app.badge.click')}</Badge>
}

export function NewBadge() {
  const { t } = useTranslation()
  return <Badge tone="new" dot="var(--accent)">{t('app.badge.new')}</Badge>
}
