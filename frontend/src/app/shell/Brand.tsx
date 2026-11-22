import { Play } from 'lucide-react'
import styles from './Sidebar.module.css'

export function Brand() {
  return <span className={styles.logo}><Play size={12} strokeWidth={2} aria-hidden /></span>
}
