import { useTranslation } from 'react-i18next'
import { Select } from './Select'
import styles from './DataTable.module.css'

export const PAGE_SIZES = [25, 50, 100, 200] as const

/** « Per page » selector of the table / grid pagers. */
export function PageSizeSelect({ value, onChange, sizes = PAGE_SIZES }: { value: number; onChange: (n: number) => void; sizes?: readonly number[] }) {
  const { t } = useTranslation()
  return (
    <span className={styles.perPage}>
      <span aria-hidden>{t('app.table.perPage')}</span>
      <span className={styles.perPageSelect}>
        <Select aria-label={t('app.table.perPage')} value={String(value)} onValueChange={(v) => onChange(Number(v))} options={sizes.map((n) => ({ value: String(n), label: String(n) }))} />
      </span>
    </span>
  )
}
