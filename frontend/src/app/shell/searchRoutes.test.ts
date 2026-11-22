import { searchItemPath } from './searchRoutes'

it('maps every search kind to its typed route', () => {
  expect(searchItemPath({ id: 'm 1', kind: 'movie', name: 'x' })).toBe('/movies/m%201')
  expect(searchItemPath({ id: 's1', kind: 'series', name: 'x' })).toBe('/series/s1')
  expect(searchItemPath({ id: 'a1', kind: 'album', name: 'x' })).toBe('/albums/a1')
  expect(searchItemPath({ id: 'ar1', kind: 'artist', name: 'x' })).toBe('/artists/ar1')
  expect(searchItemPath({ id: 'e1', kind: 'episode', name: 'x', parentId: 's1' })).toBe('/series/s1/episodes/e1')
  expect(searchItemPath({ id: 't1', kind: 'track', name: 'x', parentId: 'a1' })).toBe('/albums/a1?track=t1')
  expect(searchItemPath({ id: 't2', kind: 'track', name: 'x' })).toBeUndefined()
})
