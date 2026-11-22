import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import type { ColumnDef } from '@tanstack/react-table'
import { AppLink, Art, ButtonLink, Card, CardHeader, DataTable, EmptyState, EntityHeader, StatStrip, StatTile } from '@/ui'
import { useFormat } from '@/lib/format'
import { useEpisode, type EpisodeDetail, type EpisodeRef } from '@/api'
import { useItemMarks } from '@/api/plugin'
import { MarksCard, PlayedToEndTile, SkipsCard } from '@/app/widgets/PluginWidgets'
import { hasMarks, hasPlayedToEnd } from '@/app/widgets/plugin'
import { CompletionCell, EntitySkeleton, LibraryKicker, QueryView, UserCell } from '../library/EntityParts'
import { FunnelCard, StopsCard } from '../library/charts'
import { entityPath } from '@/lib/routes'
import { formatWhen, itemArt, useEntityCrumbs } from '../library/entity'
import styles from '../library/entity.module.css'

type Viewer = EpisodeDetail['viewers'][number]

function Episode({ seriesId, data }: { seriesId: string; data: EpisodeDetail }) {
  const { t } = useTranslation()
  const f = useFormat()
  const { item, totals } = data
  const marks = useItemMarks(item.id).data
  const showPte = hasPlayedToEnd(data.playedToEnd)
  const showSkips = (data.skips?.plays ?? 0) > 0
  const showMarks = hasMarks(marks)
  const sid = item.seriesId || seriesId
  // Neighbours may sit in another season (LAG/LEAD over the whole series).
  const code = (ref: EpisodeRef) => {
    if (ref.episode == null) return ''
    if (ref.season != null && ref.season !== item.season) return t('app.entity.episodeCode', { season: ref.season, episode: ref.episode })
    return t('app.episode.short', { n: ref.episode })
  }
  const premiere = item.premiereDate ? new Intl.DateTimeFormat(f.locale, { day: 'numeric', month: 'long', year: 'numeric' }).format(new Date(item.premiereDate)) : null
  const cols: ColumnDef<Viewer>[] = [
    { id: 'user', header: t('app.entity.col.user'), cell: ({ row }) => <UserCell user={row.original.user} /> },
    { id: 'when', header: t('app.entity.col.when'), cell: ({ row }) => <span className="num muted">{formatWhen(row.original.startedAt, f.locale)}</span> },
    { id: 'watched', header: t('app.entity.col.watched'), meta: { numeric: true }, cell: ({ row }) => f.duration(row.original.watchSeconds) },
    { id: 'completion', header: t('app.entity.col.completion'), cell: ({ row }) => <CompletionCell value={row.original.completion} width={72} /> },
    { id: 'device', header: t('app.entity.col.device'), cell: ({ row }) => <span className="ink2">{row.original.device || '—'}</span> },
  ]
  return (
    <div className={styles.stack}>
      <EntityHeader
        art={<Art title={item.name} kind="thumb" size="md" src={itemArt(item.id, 320)} seed={item.id} />}
        kicker={
          <LibraryKicker type="series" library={item.library}>
            <AppLink to={entityPath('series', sid)} className={styles.subLink}>{item.seriesName}</AppLink>
          </LibraryKicker>
        }
        title={item.name}
        meta={[item.runtimeSeconds ? f.duration(item.runtimeSeconds) : null, premiere ? t('app.episode.aired', { date: premiere }) : null].filter(Boolean)}
        actions={
          <>
            {item.prev && <ButtonLink size="sm" to={entityPath('episode', item.prev.id, sid)} aria-label={t('app.episode.prev')}><ChevronLeft size={14} aria-hidden="true" />{code(item.prev) || t('app.episode.prevShort')}</ButtonLink>}
            {item.next && <ButtonLink size="sm" to={entityPath('episode', item.next.id, sid)} aria-label={t('app.episode.next')}>{code(item.next) || t('app.episode.nextShort')}<ChevronRight size={14} aria-hidden="true" /></ButtonLink>}
          </>
        }
      />
      <Card>
        <StatStrip columns={showPte ? 5 : 4}>
          <StatTile label={t('app.entity.views')} value={f.number(totals.views)} />
          <StatTile label={t('app.entity.viewers')} value={f.number(totals.viewers)} />
          <StatTile label={t('app.metric.completionAvg')} value={totals.avgCompletion == null ? '—' : f.percent(totals.avgCompletion)} />
          {showPte && <PlayedToEndTile data={data.playedToEnd} />}
          <StatTile label={t('app.episode.pauses')} value={totals.pausedSecondsAvg == null ? '—' : f.duration(totals.pausedSecondsAvg)} foot={t('app.episode.pausesFoot')} />
        </StatStrip>
      </Card>
      <div className={styles.g2}>
        <FunnelCard funnel={data.funnel} title={t('app.episode.funnel')} subtitle={t('app.entity.funnelSub')} csvName={`episode-${item.id}-funnel`} />
        <StopsCard bins={data.stopPositions} runtimeSeconds={item.runtimeSeconds} title={t('app.episode.stops')} subtitle={t('app.episode.stopsSub')} csvName={`episode-${item.id}-stops`} />
      </div>
      {(showSkips || showMarks) && (
        <div className={showSkips && showMarks ? styles.g2 : undefined}>
          {showSkips && <SkipsCard skips={data.skips} since={data.pluginSince} />}
          {showMarks && <MarksCard data={marks} />}
        </div>
      )}
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
  useEntityCrumbs(it ? (it.episode != null ? `${t('app.episode.short', { n: it.episode })} · ${it.name}` : it.name) : undefined, it?.library, {
    series: it?.seriesName || undefined,
    season: it?.season != null ? (it.season === 0 ? t('app.series.specials') : t('app.series.season', { n: it.season })) : undefined,
    seasonNumber: it?.season != null ? String(it.season) : undefined,
  })
  return <QueryView query={q} skeleton={<EntitySkeleton />}>{(data) => <Episode seriesId={id} data={data} />}</QueryView>
}
