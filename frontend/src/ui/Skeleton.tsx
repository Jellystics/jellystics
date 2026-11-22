import styles from './Skeleton.module.css'

export function Skeleton({ width, height = 14, radius = 'sm' }: { width?: number | string; height?: number | string; radius?: 'sm' | 'md' | 'lg' }) {
  return <div className={styles.sk} style={{ width, height, borderRadius: `var(--r-${radius})` }} aria-hidden="true" />
}
