export type AIProviderCategory = 'llm' | 'tts' | 'image' | 'video'

export type AIProvider = {
  id: string
  name: string
  category: AIProviderCategory
  type: string
  model: string
  is_default: boolean
}

export type AIProviderCreate = {
  name: string
  category: AIProviderCategory
  type: string
  model: string
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

export const MODELS_BY_PROVIDER_TYPE: Record<string, { value: string; label: string }[]> = {
  anthropic: [
    { value: 'claude-opus-4-6', label: 'Claude Opus 4.6' },
    { value: 'claude-sonnet-4-6', label: 'Claude Sonnet 4.6' },
    { value: 'claude-haiku-4-5-20251001', label: 'Claude Haiku 4.5' },
  ],
  openai: [
    { value: 'gpt-5.2', label: 'GPT-5.2' },
    { value: 'gpt-5-mini', label: 'GPT-5 Mini' },
    { value: 'gpt-5-nano', label: 'GPT-5 Nano' },
    { value: 'gpt-4.1', label: 'GPT-4.1' },
    { value: 'gpt-4.1-mini', label: 'GPT-4.1 Mini' },
    { value: 'o3', label: 'o3' },
  ],
  gemini: [
    { value: 'gemini-3.1-pro-preview', label: 'Gemini 3.1 Pro' },
    { value: 'gemini-3-flash-preview', label: 'Gemini 3 Flash' },
    { value: 'gemini-2.5-pro', label: 'Gemini 2.5 Pro' },
    { value: 'gemini-2.5-flash', label: 'Gemini 2.5 Flash' },
    { value: 'gemini-2.5-flash-lite', label: 'Gemini 2.5 Flash Lite' },
  ],
  'claude-code': [
    { value: 'opus', label: 'Opus' },
    { value: 'sonnet', label: 'Sonnet' },
    { value: 'haiku', label: 'Haiku' },
  ],
  ollama: [],
  'openai-tts': [
    { value: 'gpt-4o-mini-tts', label: 'GPT-4o Mini TTS' },
    { value: 'tts-1-hd', label: 'TTS-1 HD' },
    { value: 'tts-1', label: 'TTS-1' },
  ],
  'gemini-image': [
    { value: 'gemini-3-pro-image-preview', label: 'Gemini 3 Pro Image' },
    { value: 'gemini-2.5-flash-image', label: 'Gemini 2.5 Flash Image' },
  ],
  zimage: [
    { value: 'z-image', label: 'Z-Image' },
  ],
}

export const CATEGORY_LABELS: Record<AIProviderCategory, string> = {
  llm: 'LLM',
  tts: 'TTS',
  image: 'Image',
  video: 'Video',
}

export const ALL_CATEGORIES: AIProviderCategory[] = ['llm', 'tts', 'image', 'video']
