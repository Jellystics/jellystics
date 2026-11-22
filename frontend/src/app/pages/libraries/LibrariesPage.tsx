import { useTranslation } from 'react-i18next'
import { Library } from 'lucide-react'
import { AppLink, Card, EmptyState, ErrorState, MediaTypeBadge, PageHeader, Skeleton } from '@/ui'
import { useFormat } from '@/lib/format'
import { useLibraries, type LibraryCard } from '@/api'
import { libraryFamily } from '../library/entity'
import styles from './libraries.module.css'

function LibraryTile({ lib }: { lib: LibraryCard }) {
  const { t } = useTranslation()
  const f = useFormat()
  const family = libraryFamily(lib.type)
  return (
    <AppLink to={`/libraries/${encodeURIComponent(lib.id)}`} className={styles.card}>
      <span className={styles.head}>
        <span className={styles.name}>{lib.name}</span>
        <MediaTypeBadge type={family} />
      </span>
      <span className={styles.meta}>{t(`app.libraries.items.${family}`, { count: lib.items })} · {f.bytes(lib.sizeBytes)}</span>
      <span className={styles.figures}>
        <span><b className="num">{f.duration(lib.totals.watchSeconds)}</b><small>{t('app.entity.watchTime')}</small></span>
        <span><b className="num">{f.number(lib.totals.views)}</b><small>{t('app.entity.views')}</small></span>
        <span><b className="num">{f.number(lib.totals.activeUsers)}</b><small>{t('app.entity.viewers')}</small></span>
      </span>
    </AppLink>
  )
}

export default function LibrariesPage() {
  const { t } = useTranslation()
  const q = useLibraries()
  const libs = q.data ?? []
  return (
    <div className={styles.page}>
      <PageHeader title={t('app.libraries.title')} subtitle={q.data ? t('app.libraries.subtitle', { count: libs.length }) : undefined} />
      {q.data === undefined && q.isError ? <ErrorState title={t('app.entity.error')} onRetry={() => { void q.refetch() }} />
        : q.data === undefined ? (
          <div className={styles.grid} aria-busy="true">
            {[0, 1, 2].map((i) => <Card key={i}><div className={styles.sk}><Skeleton width="50%" height={18} /><Skeleton width="35%" /><Skeleton height={36} /></div></Card>)}
          </div>
        ) : libs.length === 0 ? <EmptyState icon={<Library size={20} strokeWidth={1.6} />} title={t('app.libraries.empty')} description={t('app.libraries.emptyHint')} />
          : <div className={styles.grid}>{libs.map((l) => <LibraryTile key={l.id} lib={l} />)}</div>}
    </div>
  )
}
