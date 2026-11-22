import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Pause } from 'lucide-react'
import { Art } from './Art'
import { Avatar } from './Avatar'
import { Badge, MethodBadge, type MediaType, type PlayMethod } from './Badge'
import { Meter } from './Meter'
import { formatPosition } from '@/lib/format'
import styles from './SessionCard.module.css'

export type LiveSession = {
  id: string; title: string; subtitle: string; itemHref: string; type: MediaType
  user: { id: string; name: string }; device: string; method: PlayMethod; paused: boolean
  positionSec: number; runtimeSec: number; artSrc?: string
}

export function SessionCard({ session: s }: { session: LiveSession }) {
  const { t } = useTranslation()
  const ratio = s.runtimeSec > 0 ? s.positionSec / s.runtimeSec : 0
  return (
    <article className={styles.session}>
      <Art title={s.title} kind={s.type === 'music' ? 'square' : 'thumb'} src={s.artSrc} />
      <div className={styles.main}>
        <div className={styles.top}>
          <div className={styles.titles}>
            <Link to={s.itemHref} className={styles.title}>{s.title}</Link>
            <div className={styles.sub}>{s.subtitle}</div>
          </div>
          {s.paused && <Badge icon={<Pause size={12} aria-hidden="true" />}>{t('app.session.paused')}</Badge>}
        </div>
        <div className={styles.meta}>
          <Avatar id={s.user.id} name={s.user.name} size="sm" />
          <span>{s.user.name}</span>
          <span aria-hidden="true">·</span>
          <span>{s.device}</span>
          <MethodBadge method={s.method} />
        </div>
      </div>
      <div className={styles.progress}>
        <span>{formatPosition(s.positionSec)}</span>
        <Meter value={ratio} thin tone={s.paused ? 'muted' : 'accent'} label={t('app.session.progress')} />
        <span>{formatPosition(s.runtimeSec)}</span>
      </div>
    </article>
  )
}
