// Post-build check: dist/me.html exists, references its assets relatively (./assets/…) so it works under
// /Jellystics/Me/ and /me/, and the dashboard index.html keeps its absolute /assets/ URLs.
import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const dist = resolve(dirname(fileURLToPath(import.meta.url)), '../../dist')
const fail = (msg) => { console.error(`check-me-build: ${msg}`); process.exit(1) }
const read = (name) => { const p = resolve(dist, name); if (!existsSync(p)) fail(`${name} is missing`); return readFileSync(p, 'utf8') }

const refs = (html) => [...html.matchAll(/\b(?:src|href)="([^"]+)"/g)].map((m) => m[1])

const me = read('me.html')
const meRefs = refs(me)
if (!meRefs.some((r) => /^\.\/assets\/.+\.js$/.test(r))) fail('me.html has no ./assets/*.js script')
const absolute = meRefs.filter((r) => r.startsWith('/'))
if (absolute.length) fail(`me.html has absolute URLs: ${absolute.join(', ')}`)
for (const r of meRefs.filter((x) => x.startsWith('./'))) if (!existsSync(resolve(dist, r))) fail(`me.html references a missing file: ${r}`)

const index = read('index.html')
if (!refs(index).some((r) => /^\/assets\/.+\.js$/.test(r))) fail('index.html lost its absolute /assets/*.js script')
if (!existsSync(resolve(dist, 'locales/en/translation.json'))) fail('locales are missing from dist')

console.log(`check-me-build: ok (${meRefs.length} relative references in me.html)`)
