import { useEffect, useState } from 'react'
import { listModels } from './models'
import type { ModelInfo } from '../types'

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
