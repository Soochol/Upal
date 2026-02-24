import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface SettingsStore {
  showArchived: boolean
  setShowArchived: (value: boolean) => void
}

export const useSettingsStore = create<SettingsStore>()(
  persist(
    (set) => ({
      showArchived: false,
      setShowArchived: (value) => set({ showArchived: value }),
    }),
    { name: 'upal-settings' },
  ),
)
