import { BarChart3, Compass, Gauge, History, Home, Library, Radio, Users, type LucideIcon } from 'lucide-react'
import type { NavKey } from './crumbs'

export type NavEntry = { key: NavKey; to: string; icon: LucideIcon }
export type NavGroup = { key: 'observe' | 'explore' | 'understand'; items: NavEntry[] }

export const NAV_GROUPS: NavGroup[] = [
  { key: 'observe', items: [
    { key: 'overview', to: '/', icon: Home },
    { key: 'live', to: '/live', icon: Radio },
    { key: 'history', to: '/history', icon: History },
  ] },
  { key: 'explore', items: [
    { key: 'libraries', to: '/libraries', icon: Library },
    { key: 'users', to: '/users', icon: Users },
  ] },
  { key: 'understand', items: [
    { key: 'insights', to: '/insights', icon: BarChart3 },
    { key: 'discover', to: '/discover', icon: Compass },
    { key: 'playback', to: '/playback', icon: Gauge },
  ] },
]
export const NAV_ENTRIES = NAV_GROUPS.flatMap((g) => g.items)
