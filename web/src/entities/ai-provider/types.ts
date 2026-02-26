export type AIProviderCategory = 'llm' | 'tts' | 'image' | 'video'

export type AIProvider = {
  id: string
  name: string
  category: AIProviderCategory
  type: string
  is_default: boolean
}

export type AIProviderCreate = {
  name: string
  category: AIProviderCategory
  type: string
  api_key: string
}

export const PROVIDER_TYPES_BY_CATEGORY: Record<AIProviderCategory, { value: string; label: string }[]> = {
  llm: [
    { value: 'anthropic', label: 'Anthropic' },
    { value: 'openai', label: 'OpenAI' },
    { value: 'gemini', label: 'Google Gemini' },
    { value: 'ollama', label: 'Ollama' },
    { value: 'claude-code', label: 'Claude Code' },
  ],
  tts: [
    { value: 'openai-tts', label: 'OpenAI TTS' },
  ],
  image: [
    { value: 'gemini-image', label: 'Gemini Image' },
    { value: 'zimage', label: 'Z-Image' },
  ],
  video: [],
}

export const CATEGORY_LABELS: Record<AIProviderCategory, string> = {
  llm: 'LLM',
  tts: 'TTS',
  image: 'Image',
  video: 'Video',
}

export const ALL_CATEGORIES: AIProviderCategory[] = ['llm', 'tts', 'image', 'video']
