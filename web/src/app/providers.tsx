import type { ReactNode } from 'react'
import { ThemeProvider } from '@/shared/ui/ThemeProvider'
import { registerAllEditors } from '@/features/edit-node'

// Register node editors at module load time (one-time setup)
registerAllEditors()

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider defaultTheme="light" storageKey="upal-ui-theme">
      {children}
    </ThemeProvider>
  )
}
