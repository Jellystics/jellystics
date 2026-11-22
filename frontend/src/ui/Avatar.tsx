import styles from './Avatar.module.css'

const SLOTS = ['#6250d6', '#2a78d6', '#d55181', '#1baf7a', '#eb6834', '#52514e', '#eda100', '#e34948']
export const avatarColor = (id: string) => SLOTS[[...id].reduce((a, c) => (a * 31 + c.charCodeAt(0)) >>> 0, 7) % 8]

export function Avatar({ id, name, size = 'md', src }: { id: string; name: string; size?: 'sm' | 'md' | 'lg'; src?: string }) {
  return (
    <span className={[styles.avatar, size !== 'md' && styles[size]].filter(Boolean).join(' ')} style={{ background: avatarColor(id) }} title={name}>
      {src ? <img src={src} alt="" loading="lazy" className={styles.img} onError={(e) => { e.currentTarget.style.display = 'none' }} /> : null}
      {[...name][0]?.toUpperCase()}
    </span>
  )
}

export function AvatarStack({ users, max = 4 }: { users: { id: string; name: string }[]; max?: number }) {
  const shown = users.slice(0, max)
  const extra = users.length - shown.length
  return (
    <span className={styles.stack}>
      {shown.map((u) => <Avatar key={u.id} id={u.id} name={u.name} size="sm" />)}
      {extra > 0 && <span className={`${styles.avatar} ${styles.sm} ${styles.more}`}>+{extra}</span>}
    </span>
  )
}
