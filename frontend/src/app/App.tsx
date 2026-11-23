import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/queryClient'
import { useThemeMode } from '@/lib/themeMode'
import { Toaster, TooltipProvider } from '@/ui'
import { routes } from './routes'

const router = createBrowserRouter(routes)

export default function App() {
  useThemeMode()
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider><RouterProvider router={router} /></TooltipProvider>
      <Toaster />
    </QueryClientProvider>
  )
}
