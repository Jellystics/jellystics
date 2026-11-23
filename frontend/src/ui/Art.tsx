import { useState } from 'react'
import styles from './Art.module.css'

/** Number of --art-* placeholder tokens. */
const ART_SLOTS = 10

export function artInitials(title: string): string {
  return title.split(/\s+/).filter((w) => w.length > 2 || /^[A-Z0-9]/.test(w)).slice(0, 2).map((w) => w[0]).join('').toUpperCase()
}

export function Art({ title, kind = 'poster', size = 'sm', src, seed }: { title: string; kind?: 'poster' | 'thumb' | 'square'; size?: 'sm' | 'md' | 'lg'; src?: string; seed?: string }) {
  const [failed, setFailed] = useState(false)
  const c = `var(--art-${([...(seed ?? title)].reduce((a, ch) => a + ch.charCodeAt(0), 0) % ART_SLOTS) + 1})`
  return (
    <div
      className={[styles.art, styles[kind], size !== 'sm' && styles[size]].filter(Boolean).join(' ')}
      style={{ background: `linear-gradient(160deg, ${c}, color-mix(in srgb, ${c} 55%, var(--art-shade)))` }}
      aria-hidden="true"
    >
      {artInitials(title)}
      {src && !failed && <img className={styles.img} src={src} alt="" loading="lazy" onError={() => setFailed(true)} />}
    </div>
  )
}
