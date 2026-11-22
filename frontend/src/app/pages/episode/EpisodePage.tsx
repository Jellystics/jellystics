import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import type { ColumnDef } from '@tanstack/react-table'
import { AppLink, Art, ButtonLink, Card, CardHeader, DataTable, EmptyState, EntityHeader, MediaTypeBadge, StatStrip, StatTile } from '@/ui'
import { useFormat } from '@/lib/format'
import { useEpisode, type EpisodeDetail } from '@/api'
import { useSetCrumbLabel } from '@/app/shell/crumbs'
import { CompletionCell, EntitySkeleton, QueryView, UserCell } from '../library/EntityParts'
import { FunnelCard, StopsCard } from '../library/charts'
import { entityPath, formatWhen, itemArt } from '../library/entity'
import styles from '../library/entity.module.css'

type Viewer = EpisodeDetail['viewers'][number]

function Episode({ seriesId, data }: { seriesId: string; data: EpisodeDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  const { item, totals } = data
  const sid = item.seriesId || seriesId
  const code = (n: number | null) => (n == null ? '' : t('app.episode.short', { n }))
  const premiere = item.premiereDate ? new Intl.DateTimeFormat(f.locale, { day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(item.premiereDate)) : null
  const cols: ColumnDef<Viewer>[] = [
    { id: 'user', header: t('app.entity.col.user'), cell: ({ row }) => <UserCell user={row.original.user} /> },
    { id: 'when', header: t('app.entity.col.when'), cell: ({ row }) => <span className="num muted">{formatWhen(row.original.startedAt, f.locale)}</span> },
    { id: 'watched', header: t('app.entity.col.watched'), meta: { numeric: true }, cell: ({ row }) => f.duration(row.original.watchSeconds) },
    { id: 'completion', header: t('app.entity.col.completion'), cell: ({ row }) => <CompletionCell value={row.original.completion} width={72} /> },
    { id: 'device', header: t('app.entity.col.device'), cell: ({ row }) => <span className="ink2">{row.original.device || '—'}</span> },
  ]
  const seasonTo = item.season != null ? `${entityPath('series', sid)}?season=${item.season}` : entityPath('series', sid)
  return (
    <div className={styles.stack}>
      <EntityHeader
        art={<Art title={item.name} kind="thumb" size="md" src={itemArt(item.id, 320)} seed={item.id} />}
        kicker={
          <nav aria-label={t('app.episode.trail')} className={styles.row}>
            <MediaTypeBadge type="series" />
            <AppLink to={entityPath('series', sid)} className={styles.subLink}>{item.seriesName}</AppLink>
            {item.season != null && <><span aria-hidden="true">›</span><AppLink to={seasonTo} className={styles.subLink}>{t('app.series.season', { n: item.season })}</AppLink></>}
            {item.episode != null && <><span aria-hidden="true">›</span><span aria-current="page">{t('app.episode.number', { n: item.episode })}</span></>}
          </nav>
        }
        title={item.name}
        meta={[item.runtimeSeconds ? f.duration(item.runtimeSeconds) : null, premiere ? t('app.episode.aired', { date: premiere }) : null].filter(Boolean)}
        actions={
          <>
            {item.prevId && <ButtonLink size="sm" to={entityPath('episode', item.prevId, sid)} aria-label={t('app.episode.prev')}><ChevronLeft size={14} aria-hidden="true" />{code(item.episode != null ? item.episode - 1 : null) || t('app.episode.prevShort')}</ButtonLink>}
            {item.nextId && <ButtonLink size="sm" to={entityPath('episode', item.nextId, sid)} aria-label={t('app.episode.next')}>{code(item.episode != null ? item.episode + 1 : null) || t('app.episode.nextShort')}<ChevronRight size={14} aria-hidden="true" /></ButtonLink>}
          </>
        }
      />
      <Card>
        <StatStrip columns={4}>
          <StatTile label={t('app.entity.views')} value={f.number(totals.views)} />
          <StatTile label={t('app.entity.viewers')} value={f.number(totals.viewers)} />
          <StatTile label={t('app.episode.avgCompletion')} value={totals.avgCompletion == null ? '—' : f.percent(totals.avgCompletion)} />
          <StatTile label={t('app.episode.pauses')} value={totals.pausedSecondsAvg == null ? '—' : f.duration(totals.pausedSecondsAvg)} foot={t('app.episode.pausesFoot')} />
        </StatStrip>
      </Card>
      <div className={styles.g2}>
        <FunnelCard funnel={data.funnel} title={t('app.episode.funnel')} subtitle={t('app.entity.funnelSub')} csvName={`episode-${item.id}-funnel`} />
        <StopsCard bins={data.stopPositions} runtimeSeconds={item.runtimeSeconds} title={t('app.episode.stops')} subtitle={t('app.episode.stopsSub')} csvName={`episode-${item.id}-stops`} />
      </div>
      <Card>
        <CardHeader title={t('app.episode.viewers')} />
        <DataTable columns={cols} data={data.viewers} empty={<EmptyState title={t('app.entity.noViewers')} />} />
      </Card>
    </div>
  )
}

export default function EpisodePage() {
  const { id = '', episodeId = '' } = useParams()
  const q = useEpisode(episodeId)
  const it = q.data?.item
  const { t } = useTranslation()
  useSetCrumbLabel(it ? (it.episode != null ? `${t('app.episode.short', { n: it.episode })} · ${it.name}` : it.name) : undefined)
  return <QueryView query={q} skeleton={<EntitySkeleton />}>{(data) => <Episode seriesId={id} data={data} />}</QueryView>
}
