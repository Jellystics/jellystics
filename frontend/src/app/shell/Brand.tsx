import styles from './Sidebar.module.css'

/** Jellystics logo (same asset as the favicon). */
export function Brand() {
  return <img src="/logo-64.png" alt="" width={28} height={28} className={styles.logo} />
}
