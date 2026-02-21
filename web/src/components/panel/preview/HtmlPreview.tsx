export function HtmlPreview({ html }: { html: string }) {
  return (
    <iframe
      srcDoc={html}
      sandbox="allow-scripts allow-same-origin"
      className="w-full h-full border-0 rounded-lg bg-background"
      title="Auto-layout preview"
    />
  )
}
