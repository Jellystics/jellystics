import { entityPath, entityRoute, familyOf, mediaTypeFamily, playMethodOf, userPath } from './routes'

it('maps every entity kind to its typed route, encoding ids', () => {
  expect(entityRoute('movie', 'm 1')).toBe('/movies/m%201')
  expect(entityRoute('series', 's1')).toBe('/series/s1')
  expect(entityRoute('album', 'a1')).toBe('/albums/a1')
  expect(entityRoute('artist', 'ar1')).toBe('/artists/ar1')
  expect(entityRoute('episode', 'e1', 's 1')).toBe('/series/s%201/episodes/e1')
  expect(entityRoute('track', 't1', 'a1')).toBe('/albums/a1?track=t1')
  expect(userPath('lé a')).toBe('/users/l%C3%A9%20a')
})

it('entityRoute is undefined when unresolvable; entityPath falls back to /items/:id', () => {
  expect(entityRoute('episode', 'e1')).toBeUndefined()
  expect(entityRoute('track', 't2', null)).toBeUndefined()
  expect(entityRoute('other', 'x')).toBeUndefined()
  expect(entityPath('track', 't 2')).toBe('/items/t%202')
  expect(entityPath('episode', 'e1', 's1')).toBe('/series/s1/episodes/e1')
})

it('maps kinds, media types and play methods', () => {
  expect(familyOf('movie')).toBe('movie')
  expect(familyOf('episode')).toBe('series')
  expect(familyOf('album')).toBe('music')
  expect(familyOf('artist')).toBe('music')
  expect(mediaTypeFamily('Episode')).toBe('series')
  expect(mediaTypeFamily('Audio')).toBe('music')
  expect(mediaTypeFamily(undefined)).toBe('movie')
  expect(playMethodOf('DirectStream')).toBe('stream')
  expect(playMethodOf('DirectPlay')).toBe('direct')
  expect(playMethodOf('Unknown')).toBeNull()
  expect(playMethodOf(null)).toBeNull()
})
