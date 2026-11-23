import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ButtonLink, ErrorState, Button, MethodBadge, ViewBadge, Avatar, Art, Meter, SegBar, Breadcrumbs, EmptyState } from '@/ui'

it('Button variants and icon-only accessibility', () => {
  render(<><Button variant="primary">Sync</Button><Button iconOnly aria-label="Filtrer" icon={<svg />} /></>)
  expect(screen.getByRole('button', { name: 'Sync' })).toHaveClass('primary')
  expect(screen.getByRole('button', { name: 'Filtrer' })).toHaveClass('iconOnly')
})
it('status badges: neutral background, colored icon only', () => {
  const { container } = render(<MethodBadge method="transcode" />)
  const badge = container.firstElementChild as HTMLElement
  expect(badge).toHaveClass('badge', 'serious')
  expect(badge.querySelector('svg')).not.toBeNull()
})
it('ViewBadge labels', () => {
  render(<><ViewBadge view /><ViewBadge view={false} /></>)
  expect(screen.getByText('app.badge.view')).toBeInTheDocument()
  expect(screen.getByText('app.badge.click')).toBeInTheDocument()
})
it('Avatar color is stable per id', () => {
  const a = render(<Avatar id="lea" name="Léa" />).container.firstElementChild as HTMLElement
  const b = render(<Avatar id="lea" name="Léa" />).container.firstElementChild as HTMLElement
  expect(a.style.background).toBe(b.style.background)
  expect(a).toHaveTextContent('L')
})
it('Art falls back to initials', () => {
  render(<Art title="Station Nord" kind="poster" />)
  expect(screen.getByText('SN')).toBeInTheDocument()
})
it('Meter exposes a progressbar', () => {
  render(<Meter value={0.42} label="Complétion" />)
  expect(screen.getByRole('progressbar', { name: 'Complétion' })).toHaveAttribute('aria-valuenow', '42')
})
it('SegBar prints shares above 9 % and a legend', () => {
  render(<SegBar parts={[{ label: 'Direct', value: 68, color: 'var(--m-direct)' }, { label: 'Stream', value: 5, color: 'var(--m-stream)' }]} />)
  expect(screen.getByText(/93/)).toBeInTheDocument() // 68/73 ≈ 93 %
  expect(screen.getAllByText('Stream')).toHaveLength(1) // legend only, no in-bar label under 9 %
})
it('Breadcrumbs: last item is current page, others are links', () => {
  render(<MemoryRouter><Breadcrumbs items={[{ label: 'Séries', to: '/libraries/a' }, { label: 'Station Nord' }]} /></MemoryRouter>)
  expect(screen.getByRole('link', { name: 'Séries' })).toHaveAttribute('href', '/libraries/a')
  expect(screen.getByText('Station Nord')).toHaveAttribute('aria-current', 'page')
})
it('EmptyState', () => {
  render(<EmptyState title="Aucune lecture" description="Élargis la période" />)
  expect(screen.getByText('Aucune lecture').tagName).toBe('P')
  expect(screen.queryByRole('heading')).toBeNull()
})
it('EmptyState renders its title as a heading when given a level', () => {
  render(<EmptyState title="Page introuvable" headingLevel={1} />)
  expect(screen.getByRole('heading', { level: 1, name: 'Page introuvable' })).toBeInTheDocument()
})

it('ButtonLink renders a single anchor with button classes', () => {
  const { container } = render(<MemoryRouter><ButtonLink to="/" variant="primary" size="sm">Home</ButtonLink></MemoryRouter>)
  expect(container.querySelectorAll('a')).toHaveLength(1)
  expect(container.querySelector('button')).toBeNull()
  expect(screen.getByRole('link', { name: 'Home' })).toHaveClass('btn', 'primary', 'sm')
})
it('ErrorState is announced as an alert', () => {
  render(<ErrorState title="Oups" />)
  expect(screen.getByRole('alert')).toHaveTextContent('Oups')
})
