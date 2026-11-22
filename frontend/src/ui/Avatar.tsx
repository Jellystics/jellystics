import { useState } from 'react'
import { isJellyfinId, userImage } from '@/lib/images'
import styles from './Avatar.module.css'

/** Stable identity colour of a user: one of the 8 --avatar-* tokens. */
export const avatarColor = (id: string) => `var(--avatar-${([...id].reduce((a, c) => (a * 31 + c.charCodeAt(0)) >>> 0, 7) % 8) + 1})`

const PX = { sm: 20, md: 28, lg: 56 } as const

/**
 * User avatar: the Jellyfin profile picture when `id` is a Jellyfin user id
 * (or an explicit `src`), over a coloured initial shown while loading or when
 * the user has no picture. `src={null}` disables the picture.
 */
export function Avatar({ id, name, size = 'md', src }: { id: string; name: string; size?: 'sm' | 'md' | 'lg'; src?: string | null }) {
  const [failed, setFailed] = useState<string | null>(null)
  const url = src === undefined ? (isJellyfinId(id) ? userImage(id, PX[size] * 2) : null) : src
  return (
    <span className={[styles.avatar, size !== 'md' && styles[size]].filter(Boolean).join(' ')} style={{ background: avatarColor(id) }} title={name}>
      {[...name][0]?.toUpperCase()}
      {url && failed !== url ? <img src={url} alt="" loading="lazy" className={styles.img} onError={() => setFailed(url)} /> : null}
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
