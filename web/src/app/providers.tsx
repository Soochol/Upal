import type { ReactNode } from 'react'
import { registerAllEditors } from '@/features/edit-node'

// Register node editors at module load time (one-time setup)
registerAllEditors()

export function AppProviders({ children }: { children: ReactNode }) {
  return <>{children}</>
}
