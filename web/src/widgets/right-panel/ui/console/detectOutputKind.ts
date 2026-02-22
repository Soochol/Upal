// web/src/widgets/right-panel/ui/console/detectOutputKind.ts

export type OutputKind = 'image' | 'audio' | 'html' | 'json' | 'text'

export function detectOutputKind(output: string): OutputKind {
  const t = output.trim()
  if (!t) return 'text'

  // Image: data URI, image-extension URL, or /api/files/ serve path
  if (t.startsWith('data:image/')) return 'image'
  if (/^https?:\/\/.+\.(png|jpg|jpeg|gif|webp|svg)(\?.*)?$/i.test(t)) return 'image'
  if (/^\/api\/files\/[^/]+\/serve/.test(t)) return 'image'

  // Audio: data URI or audio-extension URL
  if (t.startsWith('data:audio/')) return 'audio'
  if (/^https?:\/\/.+\.(mp3|wav|ogg|m4a)(\?.*)?$/i.test(t)) return 'audio'

  // HTML
  const tl = t.toLowerCase()
  if (tl.startsWith('<!doctype') || tl.startsWith('<html')) return 'html'

  // JSON (only attempt if it starts with { or [)
  if (t.startsWith('{') || t.startsWith('[')) {
    try { JSON.parse(t); return 'json' } catch { /* not json */ }
  }

  return 'text'
}
