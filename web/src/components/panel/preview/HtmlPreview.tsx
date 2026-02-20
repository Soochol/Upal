export function HtmlPreview({ html }: { html: string }) {
  return (
    <iframe
      srcDoc={html}
      sandbox="allow-scripts allow-same-origin"
      className="w-full h-full border-0 rounded-lg bg-white"
      title="Auto-layout preview"
    />
  )
}
