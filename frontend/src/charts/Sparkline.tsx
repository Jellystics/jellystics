import { linePath } from './scale'

export function Sparkline({ values, width = 96, height = 28, color = 'var(--accent)' }: { values: number[]; width?: number; height?: number; color?: string }) {
  if (values.length === 0) return null
  const max = Math.max(...values), min = Math.min(...values)
  const x = (i: number) => (values.length === 1 ? width / 2 : (i * (width - 4)) / (values.length - 1) + 2)
  const y = (v: number) => height - 3 - ((v - min) / (max - min || 1)) * (height - 6)
  const p = values.map((v, i) => [x(i), y(v)] as [number, number])
  const last = p[p.length - 1]
  return (
    <svg width={width} height={height} aria-hidden="true">
      <path d={linePath(p)} fill="none" stroke={`color-mix(in srgb, ${color} 45%, var(--surface-3))`} strokeWidth={1.5} />
      <path d={linePath(p.slice(-6))} fill="none" stroke={color} strokeWidth={2} strokeLinecap="round" />
      <circle cx={last[0]} cy={last[1]} r={2.5} fill={color} />
    </svg>
  )
}
