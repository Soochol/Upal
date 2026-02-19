import { useState, useEffect } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { NodeEditor } from '@/components/editor/nodes/NodeEditor'
import { PanelConsole } from '@/components/panel/PanelConsole'
import { PanelPreview } from '@/components/panel/PanelPreview'
import { Button } from '@/components/ui/button'
import { X, Settings2, Terminal, Eye } from 'lucide-react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { NodeData } from '@/stores/workflowStore'

type RightPanelProps = {
  selectedNode: { id: string; data: NodeData } | null
  onCloseNode: () => void
}

export function RightPanel({ selectedNode, onCloseNode }: RightPanelProps) {
  const [activeTab, setActiveTab] = useState('properties')
  const isRunning = useWorkflowStore((s) => s.isRunning)

  // Auto-switch to console when running starts
  useEffect(() => {
    if (isRunning) {
      setActiveTab('console')
    }
  }, [isRunning])

  return (
    <aside className="w-80 border-l border-border bg-background flex flex-col">
      <Tabs value={activeTab} onValueChange={setActiveTab} className="flex flex-col flex-1">
        <div className="flex items-center justify-between border-b border-border px-2">
          <TabsList className="h-10 bg-transparent p-0 gap-0">
            <TabsTrigger
              value="properties"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-3 py-2 text-xs"
            >
              <Settings2 className="h-3.5 w-3.5 mr-1.5" />
              Properties
            </TabsTrigger>
            <TabsTrigger
              value="console"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-3 py-2 text-xs"
            >
              <Terminal className="h-3.5 w-3.5 mr-1.5" />
              Console
            </TabsTrigger>
            <TabsTrigger
              value="preview"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-3 py-2 text-xs"
            >
              <Eye className="h-3.5 w-3.5 mr-1.5" />
              Preview
            </TabsTrigger>
          </TabsList>
          {selectedNode && activeTab === 'properties' && (
            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onCloseNode}>
              <X className="h-4 w-4" />
            </Button>
          )}
        </div>

        <TabsContent value="properties" className="flex-1 overflow-y-auto mt-0">
          {selectedNode ? (
            <NodeEditor
              nodeId={selectedNode.id}
              data={selectedNode.data}
              onClose={onCloseNode}
              embedded
            />
          ) : (
            <div className="flex items-center justify-center h-full text-sm text-muted-foreground p-4">
              Select a node to edit its properties.
            </div>
          )}
        </TabsContent>

        <TabsContent value="console" className="flex-1 overflow-hidden mt-0">
          <PanelConsole />
        </TabsContent>

        <TabsContent value="preview" className="flex-1 overflow-hidden mt-0">
          <PanelPreview />
        </TabsContent>
      </Tabs>
    </aside>
  )
}
