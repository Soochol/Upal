import { useState, useRef, useEffect } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { NodeEditor } from '@/features/edit-node'
import { PanelPreview } from './PanelPreview'
import { PanelConsole } from './PanelConsole'
import { GroupEditor } from './GroupEditor'
import { Settings2, Terminal, Eye, PanelRightClose } from 'lucide-react'
import type { NodeData } from '@/entities/workflow'
import { useUIStore } from '@/entities/ui'
import type { Node } from '@xyflow/react'

type RightPanelProps = {
  selectedNode: Node<NodeData> | null
  onCloseNode: () => void
  onCollapse?: () => void
  onTogglePanel?: () => void
}

const tabs = [
  { value: 'properties', label: 'Properties', icon: Settings2 },
  { value: 'console', label: 'Console', icon: Terminal },
  { value: 'preview', label: 'Preview', icon: Eye },
] as const

export function RightPanel({ selectedNode, onCloseNode, onCollapse, onTogglePanel }: RightPanelProps) {
  const [activeTab, setActiveTab] = useState('properties')

  // Ref: read latest activeTab inside effects without adding dependencies
  const activeTabRef = useRef(activeTab)
  activeTabRef.current = activeTab

  // ── Node selection → switch to Properties tab; deselection on Properties → signal collapse ──
  const prevNodeIdRef = useRef<string | null>(selectedNode?.id ?? null)
  useEffect(() => {
    if (selectedNode) {
      setActiveTab('properties')
    } else if (prevNodeIdRef.current !== null && activeTabRef.current === 'properties') {
      // Node was deselected while on Properties tab — ask parent to hide the panel
      onCollapse?.()
    }
    prevNodeIdRef.current = selectedNode?.id ?? null
  }, [selectedNode?.id, onCollapse])

  // ── Force Preview tab (from Ctrl+Enter or store signal) ──
  useEffect(() => {
    const unsub = useUIStore.subscribe(
      (state, prevState) => {
        if (state.forcePreviewTab && !prevState.forcePreviewTab) {
          setActiveTab('preview')
          useUIStore.getState().setForcePreviewTab(false)
        }
      },
    )
    return unsub
  }, [])

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab} className="flex flex-col flex-1 min-h-0 gap-0">
      <div className="flex items-center border-b border-border px-1">
        <TabsList className="h-10 bg-transparent p-0 gap-0 flex-1">
          {tabs.map((tab) => {
            const Icon = tab.icon
            return (
              <TabsTrigger
                key={tab.value}
                value={tab.value}
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-3 py-2 flex items-center gap-1.5 text-xs font-medium"
              >
                <Icon className="h-3.5 w-3.5 shrink-0" />
                {tab.label}
              </TabsTrigger>
            )
          })}
        </TabsList>
        {onTogglePanel && (
          <button
            onClick={onTogglePanel}
            className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer shrink-0 mr-1"
            title="Close Panel"
          >
            <PanelRightClose className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Properties: flex-fill so prompt fields expand to fill space */}
      <TabsContent value="properties" className="flex-1 min-h-0 flex flex-col mt-0">
        {selectedNode && selectedNode.type === 'groupNode' ? (
          <GroupEditor groupId={selectedNode.id} data={selectedNode.data as NodeData} onClose={onCloseNode} />
        ) : selectedNode ? (
          <NodeEditor
            nodeId={selectedNode.id}
            data={selectedNode.data as NodeData}
            onClose={onCloseNode}
            embedded
          />
        ) : (
          <div className="flex items-center justify-center h-32 text-xs text-muted-foreground p-3">
            Select a node to edit its properties.
          </div>
        )}
      </TabsContent>

      {/* Other tabs: flex-1 to fill remaining space */}
      <TabsContent value="console" className="flex-1 min-h-0 overflow-hidden mt-0">
        <PanelConsole />
      </TabsContent>

      <TabsContent value="preview" className="flex-1 min-h-0 overflow-hidden mt-0">
        <PanelPreview />
      </TabsContent>

    </Tabs>
  )
}
