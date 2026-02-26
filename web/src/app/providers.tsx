import { type ReactNode, useEffect } from 'react'
import { ThemeProvider } from '@/shared/ui/ThemeProvider'
import { registerAllEditors } from '@/features/edit-node'
import { useAuthStore } from '@/entities/auth'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

// Register node editors at module load time (one-time setup)
registerAllEditors()

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 1000 * 60 * 5, // 5 minutes
    },
  },
})

export function AppProviders({ children }: { children: ReactNode }) {
  useEffect(() => {
    useAuthStore.getState().init()
  }, [])

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="light" storageKey="upal-ui-theme">
        {children}
      </ThemeProvider>
    </QueryClientProvider>
  )
}
