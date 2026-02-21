import { useState, useRef, useEffect } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { NodeEditor } from '@/components/editor/nodes/NodeEditor'
import { PanelPreview } from '@/components/panel/PanelPreview'
import { PanelConsole } from '@/components/panel/PanelConsole'
import { GroupEditor } from '@/components/panel/GroupEditor'
import { AIChatEditor } from '@/components/panel/AIChatEditor'
import { Settings2, Terminal, Eye } from 'lucide-react'
import type { NodeData } from '@/stores/workflowStore'
import { useUIStore } from '@/stores/uiStore'
import { useResizeDrag } from '@/hooks/useResizeDrag'
import type { Node } from '@xyflow/react'

type RightPanelProps = {
  selectedNode: Node<NodeData> | null
  onCloseNode: () => void
}

const tabs = [
  { value: 'properties', label: 'Properties', icon: Settings2 },
  { value: 'console', label: 'Console', icon: Terminal },
  { value: 'preview', label: 'Preview', icon: Eye },
] as const

const DEFAULT_WIDTH = 512
const MIN_WIDTH = 280
const MAX_WIDTH = 800

export function RightPanel({ selectedNode, onCloseNode }: RightPanelProps) {
  const [activeTab, setActiveTab] = useState('properties')
  const [expanded, setExpanded] = useState(true)
  const { size: width, handleMouseDown } = useResizeDrag({
    direction: 'horizontal',
    min: MIN_WIDTH,
    max: MAX_WIDTH,
    initial: DEFAULT_WIDTH,
  })

  // Refs: read latest state inside effects without adding dependencies
  const activeTabRef = useRef(activeTab)
  activeTabRef.current = activeTab
  const expandedRef = useRef(expanded)
  expandedRef.current = expanded

  // ── State machine: auto-expand / auto-collapse ──
  // - Node selected  → expand (switch to Properties only if was collapsed)
  // - Node deselected + on Properties → collapse
  // - Node deselected + on Logs/Data/… → stay expanded
  const prevNodeIdRef = useRef<string | null>(selectedNode?.id ?? null)
  useEffect(() => {
    if (selectedNode) {
      setActiveTab('properties')
      setExpanded(true)
    } else if (prevNodeIdRef.current !== null && activeTabRef.current === 'properties') {
      // Only auto-collapse on deselection (selected → null), not on initial mount
      setExpanded(false)
    }
    prevNodeIdRef.current = selectedNode?.id ?? null
  }, [selectedNode?.id])

  // ── Force Preview tab (from Ctrl+Enter or store signal) ──
  useEffect(() => {
    const unsub = useUIStore.subscribe(
      (state, prevState) => {
        if (state.forcePreviewTab && !prevState.forcePreviewTab) {
          setActiveTab('preview')
          setExpanded(true)
          useUIStore.getState().setForcePreviewTab(false)
        }
      },
    )
    return unsub
  }, [])

  const selectedNodeId = selectedNode?.id ?? null
  const showAIChat = selectedNode && selectedNode.type !== 'groupNode'

  // ── Collapsed: vertical icon strip (like VS Code activity bar) ──
  if (!expanded) {
    return (
      <aside className="border-l border-border bg-background flex flex-col items-center pt-1.5 w-10 shrink-0">
        <TooltipProvider>
          {tabs.map((tab) => {
            const Icon = tab.icon
            return (
              <Tooltip key={tab.value}>
                <TooltipTrigger asChild>
                  <button
                    onClick={() => {
                      setActiveTab(tab.value)
                      setExpanded(true)
                    }}
                    className="p-2 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                  >
                    <Icon className="h-3.5 w-3.5" />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="left">{tab.label}</TooltipContent>
              </Tooltip>
            )
          })}
        </TooltipProvider>
      </aside>
    )
  }

  // ── Expanded: full panel ──
  return (
    <aside className="border-l border-border bg-background flex flex-col relative" style={{ width, maxWidth: '45%' }}>
      {/* Resize handle */}
      <div
        onMouseDown={handleMouseDown}
        className="absolute left-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 transition-colors z-10"
      />
      <Tabs value={activeTab} onValueChange={setActiveTab} className="flex flex-col flex-1 min-h-0 gap-0">
        <div className="flex items-center justify-between border-b border-border px-1">
          <TooltipProvider>
            <TabsList className="h-10 bg-transparent p-0 gap-0">
              {tabs.map((tab) => {
                const Icon = tab.icon
                return (
                  <Tooltip key={tab.value}>
                    <TooltipTrigger asChild>
                      <TabsTrigger
                        value={tab.value}
                        className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-2 py-2"
                      >
                        <Icon className="h-3.5 w-3.5" />
                      </TabsTrigger>
                    </TooltipTrigger>
                    <TooltipContent side="bottom">{tab.label}</TooltipContent>
                  </Tooltip>
                )
              })}
            </TabsList>
          </TooltipProvider>
          {/* Close button is inside NodeEditor header for properties tab */}
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
          <PanelConsole selectedNodeId={selectedNodeId} />
        </TabsContent>

        <TabsContent value="preview" className="flex-1 min-h-0 overflow-hidden mt-0">
          <PanelPreview />
        </TabsContent>

        {/* AI Assistant — pinned to bottom of panel */}
        {showAIChat && (
          <div className="mt-auto shrink-0 border-t border-border bg-background">
            <AIChatEditor nodeId={selectedNode.id} data={selectedNode.data as NodeData} />
          </div>
        )}
      </Tabs>
    </aside>
  )
}
