import { Button } from '@/shared/ui/button'
import { Download, ExternalLink } from 'lucide-react'

export function ImagePreview({ dataURI, workflowName }: { dataURI: string; workflowName: string }) {
  return (
    <div className="space-y-1.5">
      <img
        src={dataURI}
        alt="Generated image"
        className="max-w-full rounded-lg shadow-sm border border-border"
      />
      <div className="flex gap-1">
        <Button
          variant="ghost"
          size="sm"
          className="h-6 px-2 text-[10px]"
          onClick={() => {
            const a = document.createElement('a')
            a.href = dataURI
            a.download = `${workflowName || 'generated'}-image.png`
            a.click()
          }}
        >
          <Download className="h-3 w-3 mr-1" />
          Save
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="h-6 px-2 text-[10px]"
          onClick={() => window.open(dataURI, '_blank')}
        >
          <ExternalLink className="h-3 w-3 mr-1" />
          Open
        </Button>
      </div>
    </div>
  )
}
