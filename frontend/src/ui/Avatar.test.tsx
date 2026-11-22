import { fireEvent, render } from '@testing-library/react'
import { expect, it } from 'vitest'
import { Avatar } from './Avatar'

const JF = '5f1e0c3a9b2d4e6f8a7b6c5d4e3f2a1b'

it('loads the Jellyfin profile picture for a Jellyfin user id', () => {
  const { container } = render(<Avatar id={JF} name="Léa" />)
  expect(container.querySelector('img')?.getAttribute('src')).toBe(`/proxy/Users/Images/Primary/?id=${JF}&fillWidth=56&quality=90`)
  expect(container.textContent).toBe('L')
})

it('keeps the initial when the picture fails or the id is not a Jellyfin id', () => {
  const { container, rerender } = render(<Avatar id={JF} name="Léa" />)
  fireEvent.error(container.querySelector('img')!)
  expect(container.querySelector('img')).toBeNull()
  rerender(<Avatar id="guest" name="Guest" />)
  expect(container.querySelector('img')).toBeNull()
  rerender(<Avatar id={JF} name="Léa" src={null} />)
  expect(container.querySelector('img')).toBeNull()
})
