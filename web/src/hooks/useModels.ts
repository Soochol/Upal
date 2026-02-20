import { useEffect, useState } from 'react'
import { listModels, type ModelInfo } from '@/lib/api'

let cachedModels: ModelInfo[] | null = null

export function useModels() {
  const [models, setModels] = useState<ModelInfo[]>(cachedModels ?? [])
  useEffect(() => {
    if (cachedModels) return
    listModels()
      .then((m) => {
        cachedModels = m
        setModels(m)
      })
      .catch(() => setModels([]))
  }, [])
  return models
}
