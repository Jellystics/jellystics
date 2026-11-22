import { render, screen } from '@testing-library/react'
import { expect, it, vi } from 'vitest'
import { toast } from '@/ui'
import App from './App'

vi.mock('@/lib/socket', () => ({ default: { connect: vi.fn(), disconnect: vi.fn() }, useSocketEvent: vi.fn() }))
vi.mock('./routes', async () => {
  const { AppShell } = await import('./shell/AppShell')
  return { routes: [{ element: <AppShell />, children: [{ path: '*', element: <p>page</p> }] }] }
})

it('mounts a single toast region: one toast renders exactly one node', async () => {
  render(<App />)
  await screen.findByText('page')
  toast.info('hello once')
  expect(await screen.findAllByText('hello once')).toHaveLength(1)
  expect(document.querySelectorAll('[role="status"]')).toHaveLength(1)
})
