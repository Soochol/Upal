import { useState, useEffect } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Database, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

function formatKey(key: string): string {
  if (key.startsWith('__user_input__')) {
    return `User Input: ${key.replace('__user_input__', '')}`
  }
  return key
}

type PanelDataProps = {
  selectedNodeId: string | null
}

export function PanelData({ selectedNodeId }: PanelDataProps) {
  const sessionState = useWorkflowStore((s) => s.sessionState)
  const entries = Object.entries(sessionState)

  const [openSections, setOpenSections] = useState<Record<string, boolean>>({})

  // Auto-expand selected node's section when selectedNodeId changes
  useEffect(() => {
    if (selectedNodeId && selectedNodeId in sessionState) {
      setOpenSections((prev) => ({ ...prev, [selectedNodeId]: true }))
    }
  }, [selectedNodeId, sessionState])

  const toggleSection = (key: string) => {
    setOpenSections((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  if (entries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground p-6">
        <Database className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm text-center">Run a workflow to see data here.</p>
      </div>
    )
  }

  return (
    <ScrollArea className="h-full">
      <div className="p-3 space-y-1">
        {entries.map(([key, value]) => {
          const isSelected = selectedNodeId === key
          const isOpen = openSections[key] ?? false

          return (
            <Collapsible
              key={key}
              open={isOpen}
              onOpenChange={() => toggleSection(key)}
            >
              <CollapsibleTrigger
                className={cn(
                  'flex items-center gap-2 w-full rounded-md px-2.5 py-2 text-left text-sm transition-colors hover:bg-muted/50',
                  isSelected && 'ring-2 ring-primary/50 bg-primary/5',
                )}
              >
                <ChevronRight
                  className={cn(
                    'h-3.5 w-3.5 shrink-0 text-muted-foreground transition-transform duration-200',
                    isOpen && 'rotate-90',
                  )}
                />
                <span className="font-medium truncate">{formatKey(key)}</span>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <div className="ml-6 mr-2 mt-1 mb-2">
                  <pre
                    className={cn(
                      'rounded-md border border-border bg-muted/30 p-3 text-xs font-mono whitespace-pre-wrap break-all overflow-hidden',
                      isSelected && 'border-primary/30',
                    )}
                  >
                    {typeof value === 'string' ? value : JSON.stringify(value, null, 2)}
                  </pre>
                </div>
              </CollapsibleContent>
            </Collapsible>
          )
        })}
      </div>
    </ScrollArea>
  )
}
