import { fireEvent, render, screen } from '@testing-library/react'
import { expect, it, vi } from 'vitest'
import { SearchMenu } from './SearchMenu'

const options = ['Céline', 'Léa', 'Marc', 'Hugo'].map((n, i) => ({ value: `u${i}`, label: n }))

function setup() {
  const onSelect = vi.fn()
  render(<SearchMenu options={options} onSelect={onSelect} searchPlaceholder="Search" emptyLabel="No results" trigger={<button type="button">open</button>} />)
  fireEvent.click(screen.getByRole('button', { name: 'open' }))
  return onSelect
}

it('filters options accent-insensitively and selects with the keyboard', () => {
  const onSelect = setup()
  const input = screen.getByRole('combobox', { name: 'Search' })
  fireEvent.change(input, { target: { value: 'lea' } })
  expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(['Léa'])
  fireEvent.keyDown(input, { key: 'Enter' })
  expect(onSelect).toHaveBeenCalledWith('u1')
})

it('shows the empty label and moves the highlight with arrows', () => {
  const onSelect = setup()
  const input = screen.getByRole('combobox', { name: 'Search' })
  fireEvent.keyDown(input, { key: 'ArrowDown' })
  fireEvent.keyDown(input, { key: 'Enter' })
  expect(onSelect).toHaveBeenCalledWith('u1')
  fireEvent.click(screen.getByRole('button', { name: 'open' }))
  fireEvent.change(screen.getByRole('combobox', { name: 'Search' }), { target: { value: 'zzz' } })
  expect(screen.getByText('No results')).toBeInTheDocument()
})
