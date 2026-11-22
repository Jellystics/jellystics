import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

type Tree = { [k: string]: string | Tree }
const load = (lng: string): Tree => JSON.parse(readFileSync(resolve(__dirname, `../../public/locales/${lng}/translation.json`), 'utf8'))
const flatten = (o: Tree, p = ''): string[] =>
  Object.entries(o).flatMap(([k, v]) => (typeof v === 'object' ? flatten(v, `${p}${k}.`) : [`${p}${k}`])).sort()

describe('locales', () => {
  const en = flatten(load('en'))

  it.each(['fr', 'es'])('%s has exactly the same keys as en', (lng) => {
    const keys = flatten(load(lng))
    expect({ missing: en.filter((k) => !keys.includes(k)), extra: keys.filter((k) => !en.includes(k)) }).toEqual({ missing: [], extra: [] })
  })

  it.each(['en', 'fr', 'es'])('%s has no empty values', (lng) => {
    const tree = load(lng)
    const empty = flatten(tree).filter((k) => k.split('.').reduce<string | Tree>((n, part) => (n as Tree)[part], tree) === '')
    expect(empty).toEqual([])
  })
})
