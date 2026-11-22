import { useTranslation } from 'react-i18next'
import { Radio } from 'lucide-react'
import { useLiveSessions } from '@/api/live'
import { AppLink, Badge, Card, CardBody, CardFooter, CardHeader, EmptyState, SessionCard, Skeleton } from '@/ui'
import { toLiveSession } from '../live/toLiveSession'
import l from './layout.module.css'

/** « En direct » card of the overview: the first two live sessions, link to /live. */
export function LiveCard() {
  const { t } = useTranslation()
  const { data, isPending } = useLiveSessions()
  const sessions = data ?? []
  const transcodes = sessions.filter((s) => s.PlayState?.PlayMethod === 'Transcode').length
  return (
    <Card>
      <CardHeader
        title={t('app.overview.live.title')}
        subtitle={t('app.overview.live.subtitle', { count: sessions.length, transcodes })}
        action={sessions.length > 0 ? <Badge tone="critical" icon={<Radio size={12} aria-hidden="true" />}>{t('app.overview.live.badge')}</Badge> : undefined}
      />
      <CardBody>
        {isPending ? (
          <div className={l.stack}><Skeleton height={72} radius="md" /><Skeleton height={72} radius="md" /></div>
        ) : sessions.length === 0 ? (
          <EmptyState icon={<Radio size={20} strokeWidth={1.6} />} title={t('app.overview.live.empty')} />
        ) : (
          <div className={l.stack}>{sessions.slice(0, 2).map((s) => <SessionCard key={s.Id} session={toLiveSession(s, t)} />)}</div>
        )}
      </CardBody>
      <CardFooter>
        <span>{t('app.overview.live.realtime')}</span>
        <AppLink to="/live">{t('app.overview.seeAll')}</AppLink>
      </CardFooter>
    </Card>
  )
}
