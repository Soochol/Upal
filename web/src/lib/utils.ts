import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"
import type { ModelInfo } from './api'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function groupModelsByProvider(models: ModelInfo[]): Record<string, ModelInfo[]> {
  return models.reduce<Record<string, ModelInfo[]>>((acc, m) => {
    ;(acc[m.provider] ??= []).push(m)
    return acc
  }, {})
}
