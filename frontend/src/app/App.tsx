import { useMemo } from 'react'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/queryClient'
import { useThemeMode } from '@/lib/themeMode'
import { Toaster } from '@/ui'
import { routes } from './routes'

export default function App() {
  useThemeMode()
  const router = useMemo(() => createBrowserRouter(routes), [])
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <Toaster />
    </QueryClientProvider>
  )
}
