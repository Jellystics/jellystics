import { useState } from 'react'
import styles from './Art.module.css'

const ART = ['#3b4a63', '#5b3f4f', '#2f5a54', '#5a4a2f', '#3f3b5e', '#4d5a2f', '#5e3b3b', '#2f4f5e', '#4a3b5e', '#3d5a45']

export function artInitials(title: string): string {
  return title.split(/\s+/).filter((w) => w.length > 2 || /^[A-Z0-9]/.test(w)).slice(0, 2).map((w) => w[0]).join('').toUpperCase()
}

export function Art({ title, kind = 'poster', size = 'sm', src, seed }: { title: string; kind?: 'poster' | 'thumb' | 'square'; size?: 'sm' | 'md' | 'lg'; src?: string; seed?: string }) {
  const [failed, setFailed] = useState(false)
  const c = ART[[...(seed ?? title)].reduce((a, ch) => a + ch.charCodeAt(0), 0) % ART.length]
  return (
    <div
      className={[styles.art, styles[kind], size !== 'sm' && styles[size]].filter(Boolean).join(' ')}
      style={{ background: `linear-gradient(160deg, ${c}, color-mix(in srgb, ${c} 55%, #000))` }}
      aria-hidden="true"
    >
      {artInitials(title)}
      {src && !failed && <img className={styles.img} src={src} alt="" loading="lazy" onError={() => setFailed(true)} />}
    </div>
  )
}
