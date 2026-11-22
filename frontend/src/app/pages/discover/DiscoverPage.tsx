import { useTranslation } from 'react-i18next'
import { itemImage } from '@/lib/images'
import { useDiscoverAbandoned, useDiscoverAdded, useDiscoverStorage, useDiscoverUnwatched } from '@/api/analytics'
import { ChartFrame, Scatter } from '@/charts'
import { AppLink, Art, Card, CardBody, CardHeader, ErrorState, MediaTypeBadge, Meter, PageHeader, Skeleton, StatStrip, StatTile } from '@/ui'
import { useFormat } from '@/lib/format'
import { QueryCard } from '../_shared/shared'
import { entityPath, familyOf, formatBytes } from '../_shared/analyticsFormat'
import styles from '../_shared/analytics.module.css'

const GB = 1_000_000_000

export default function DiscoverPage() {
  const { t } = useTranslation()
  const f = useFormat()
  const unwatched = useDiscoverUnwatched()
  const abandoned = useDiscoverAbandoned()
  const added = useDiscoverAdded()
  const storage = useDiscoverStorage()

  const tiles = [unwatched, abandoned, added]
  const tilesFailed = tiles.some((q) => q.isError && q.data === undefined)
  const tilesLoading = tiles.some((q) => q.data === undefined) && !tilesFailed
  const bytes = (b: number) => formatBytes(b, f.locale)

  return (
    <div className={styles.page}>
      <PageHeader title={t('app.discover.title')} subtitle={t('app.discover.subtitle')} />
      <Card>
        {tilesFailed ? (
          <CardBody><ErrorState title={t('app.error.generic')} onRetry={() => tiles.forEach((q) => { if (q.isError) q.refetch() })} /></CardBody>
        ) : tilesLoading ? (
          <CardBody><div aria-busy="true" className={styles.tilesSkeleton}><Skeleton height={64} radius="md" /></div></CardBody>
        ) : (
          <StatStrip columns={3}>
            <StatTile
              label={t('app.discover.unwatched.title')} value={f.number(unwatched.data?.count ?? 0)}
              foot={unwatched.data && unwatched.data.catalog > 0 ? t('app.discover.unwatched.foot', {
                share: f.percent(unwatched.data.count / unwatched.data.catalog), size: bytes(unwatched.data.sizeBytes),
              }) : undefined}
            />
            <StatTile label={t('app.discover.abandoned.title')} value={f.number(abandoned.data?.count ?? 0)} foot={t('app.discover.abandoned.foot')} />
            <StatTile
              label={t('app.discover.added.title')} value={f.number(added.data?.count ?? 0)}
              foot={t('app.discover.added.foot', { count: added.data?.seen ?? 0 })}
            />
          </StatStrip>
        )}
      </Card>

      <div className={styles.grid}>
        <QueryCard title={t('app.discover.unwatched.title')} query={unwatched} isEmpty={(d) => d.items.length === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.discover.unwatched.title')} subtitle={t('app.discover.unwatched.subtitle')} />
              <CardBody flush>
                <table className={styles.table}>
                  <thead className={styles.srHead}>
                    <tr>
                      <th scope="col"><span className="sr-only">{t('app.insights.content')}</span></th>
                      <th scope="col"><span className="sr-only">{t('app.discover.unwatched.kind')}</span></th>
                    </tr>
                  </thead>
                  <tbody>
                    {d.items.map((it) => {
                      const to = entityPath(it.kind, it.id)
                      return (
                        <tr key={it.id}>
                          <td>
                            <div className={styles.media}>
                              <Art title={it.name} seed={it.id} src={itemImage(it.id)} kind={it.kind === 'album' ? 'square' : 'poster'} />
                              <div>
                                {to ? <AppLink className={styles.title} to={to}>{it.name}</AppLink> : <span className={styles.title}>{it.name}</span>}
                                <div className={styles.sub}>
                                  {it.dateAdded ? t('app.discover.unwatched.added', { when: f.relative(new Date(`${it.dateAdded}T12:00:00`)) }) : t('app.discover.unwatched.addedUnknown')}
                                  {it.sizeBytes > 0 && ` · ${bytes(it.sizeBytes)}`}
                                </div>
                              </div>
                            </div>
                          </td>
                          <td className={styles.num}><MediaTypeBadge type={familyOf(it.kind)} /></td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.discover.abandoned.title')} query={abandoned} isEmpty={(d) => d.items.length === 0}>
          {(d) => (
            <Card>
              <CardHeader title={t('app.discover.abandoned.title')} subtitle={t('app.discover.abandoned.subtitle')} />
              <CardBody flush>
                <table className={styles.table}>
                  <thead className={styles.srHead}>
                    <tr>
                      <th scope="col"><span className="sr-only">{t('app.insights.content')}</span></th>
                      <th scope="col"><span className="sr-only">{t('app.discover.abandoned.progressCol')}</span></th>
                    </tr>
                  </thead>
                  <tbody>
                    {d.items.map((it) => (
                      <tr key={`${it.kind}/${it.id}/${it.user.id}`}>
                        <td>
                          <div className={styles.media}>
                            <Art title={it.name} seed={it.id} src={itemImage(it.id)} />
                            <div>
                              <AppLink className={styles.title} to={entityPath(it.kind, it.id) ?? '/'}>{it.name}</AppLink>
                              <div className={styles.sub}>
                                <AppLink to={`/users/${encodeURIComponent(it.user.id)}`}>{it.user.name}</AppLink>
                                {' · '}
                                {it.kind === 'series'
                                  ? t('app.discover.abandoned.episodes', { viewed: it.episodesViewed ?? 0, total: it.episodesTotal ?? 0 })
                                  : t('app.discover.abandoned.stoppedAt', { position: f.position(it.positionSeconds ?? 0) })}
                                {' · '}
                                {f.relative(new Date(it.lastPlayed))}
                              </div>
                            </div>
                          </div>
                        </td>
                        <td className={styles.meterCell}>
                          <Meter value={it.progress} tone="serious" thin label={t('app.discover.abandoned.progress', { value: f.percent(it.progress) })} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </CardBody>
            </Card>
          )}
        </QueryCard>

        <QueryCard title={t('app.discover.storage.title')} query={storage} wide isEmpty={(d) => d.length === 0}>
          {(d) => (
            <ChartFrame
              title={t('app.discover.storage.title')} subtitle={t('app.discover.storage.subtitle')} csvName="storage-vs-views"
              legend={[{ label: t('app.discover.storage.neverViewed'), color: 'var(--serious)' }, { label: t('app.discover.storage.viewed'), color: 'var(--accent)' }]}
              table={{ columns: [t('app.insights.content'), t('app.discover.storage.sizeGbColumn'), t('app.chart.views')], rows: d.map((p) => [p.name, Math.round((p.sizeBytes / GB) * 100) / 100, p.views]) }}
            >
              <Scatter
                points={d.map((p) => ({ label: p.name, x: p.sizeBytes / GB, y: p.views, color: p.views === 0 ? 'var(--serious)' : 'var(--accent)' }))}
                xLabel={t('app.discover.storage.sizeGb')} yLabel={t('app.chart.views')}
                xFormat={(v) => bytes(v * GB)} note={t('app.discover.storage.quadrant')}
              />
            </ChartFrame>
          )}
        </QueryCard>
      </div>
    </div>
  )
}
