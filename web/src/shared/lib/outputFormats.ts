import type { ComponentType } from 'react'
import type { LucideIcon } from 'lucide-react'
import { Download, ExternalLink } from 'lucide-react'

// ── Domain Types ──

export type OutputFormatId = 'html' | 'md'

export type FormatAction = {
  id: string
  label: string
  icon: LucideIcon
  handler: (content: string, workflowName: string) => void
}

export type ResultViewProps = {
  content: string
  workflowName: string
}

type EditorField = 'model' | 'system_prompt' | 'prompt'

export type OutputFormatDef = {
  id: OutputFormatId
  label: string
  /** Fields shown in OutputNodeEditor when this format is active */
  editorFields: EditorField[]
  /** Action buttons shown in the results panel */
  actions: FormatAction[]
  /** Auto-detect this format from raw content (fallback for legacy workflows) */
  detect: (content: string) => boolean
  /** Lazy-loaded result view component */
  ResultView: () => Promise<{ default: ComponentType<ResultViewProps> }>
}

// ── Action Handlers ──

function downloadFile(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

function openInBrowser(content: string) {
  const blob = new Blob([content], { type: 'text/html' })
  const url = URL.createObjectURL(blob)
  window.open(url, '_blank')
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

// ── Format Registry ──

export const OUTPUT_FORMATS: Record<OutputFormatId, OutputFormatDef> = {
  html: {
    id: 'html',
    label: 'HTML',
    editorFields: ['model', 'system_prompt', 'prompt'],
    actions: [
      {
        id: 'open-tab',
        label: 'Open in tab',
        icon: ExternalLink,
        handler: (content) => openInBrowser(content),
      },
      {
        id: 'save-html',
        label: 'Save',
        icon: Download,
        handler: (content, name) => downloadFile(content, `${name || 'output'}.html`, 'text/html'),
      },
    ],
    detect: (s) => {
      const t = s.trimStart().toLowerCase()
      return t.startsWith('<!doctype') || t.startsWith('<html')
    },
    ResultView: () => import('@/widgets/right-panel/ui/preview/HtmlResultView'),
  },
  md: {
    id: 'md',
    label: 'Markdown',
    editorFields: ['prompt'],
    actions: [
      {
        id: 'save-md',
        label: 'Save',
        icon: Download,
        handler: (content, name) => downloadFile(content, `${name || 'output'}.md`, 'text/markdown'),
      },
    ],
    detect: () => false, // MD cannot be auto-detected; requires explicit config
    ResultView: () => import('@/widgets/right-panel/ui/preview/MarkdownResultView'),
  },
}

/** Resolve format from config hint, with auto-detect fallback for legacy workflows. */
export function resolveFormat(
  formatHint: string | undefined,
  content: string,
): OutputFormatDef {
  if (formatHint && formatHint in OUTPUT_FORMATS) {
    return OUTPUT_FORMATS[formatHint as OutputFormatId]
  }
  // Legacy: no output_format in config → auto-detect from content
  for (const fmt of Object.values(OUTPUT_FORMATS)) {
    if (fmt.detect(content)) return fmt
  }
  return OUTPUT_FORMATS.md // default fallback: plain text / markdown
}
