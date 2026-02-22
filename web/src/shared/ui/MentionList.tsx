import { forwardRef, useEffect, useImperativeHandle, useState } from 'react'
import type { ComponentType } from 'react'
import { Bot } from 'lucide-react'
import { getNodeDefinition } from '@/entities/node'
import type { NodeType } from '@/entities/node'

export type MentionItem = {
  id: string
  label: string
  nodeType: string
}

export type MentionListRef = {
  onKeyDown: (event: KeyboardEvent) => boolean
}

type MentionListProps = {
  items: MentionItem[]
  command: (item: MentionItem) => void
}

const colorMap: Record<string, string> = {
  input: 'text-node-input',
  agent: 'text-node-agent',
  output: 'text-node-output',
}

export const MentionList = forwardRef<MentionListRef, MentionListProps>(
  ({ items, command }, ref) => {
    const [selectedIndex, setSelectedIndex] = useState(0)

    useEffect(() => {
      setSelectedIndex(0)
    }, [items])

    useImperativeHandle(ref, () => ({
      onKeyDown: (event: KeyboardEvent) => {
        if (event.key === 'ArrowUp') {
          setSelectedIndex((prev) => (prev + items.length - 1) % items.length)
          return true
        }
        if (event.key === 'ArrowDown') {
          setSelectedIndex((prev) => (prev + 1) % items.length)
          return true
        }
        if (event.key === 'Enter') {
          const item = items[selectedIndex]
          if (item) command(item)
          return true
        }
        return false
      },
    }))

    if (items.length === 0) {
      return (
        <div className="rounded-lg border border-border bg-popover p-3 shadow-md">
          <p className="text-xs text-muted-foreground">No upstream nodes connected</p>
        </div>
      )
    }

    return (
      <div className="rounded-lg border border-border bg-popover shadow-md overflow-hidden min-w-[200px]">
        {items.map((item, index) => {
          let Icon: ComponentType<{ className?: string }> = Bot
          try { Icon = getNodeDefinition(item.nodeType as NodeType).icon } catch { /* unknown type */ }
          const color = colorMap[item.nodeType] || 'text-muted-foreground'
          return (
            <button
              key={item.id}
              className={`flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors ${
                index === selectedIndex
                  ? 'bg-accent text-accent-foreground'
                  : 'hover:bg-accent/50'
              }`}
              onClick={() => command(item)}
              onMouseEnter={() => setSelectedIndex(index)}
              type="button"
            >
              <Icon className={`h-3.5 w-3.5 shrink-0 ${color}`} />
              <span className="truncate">{item.label}</span>
              <span className="ml-auto text-[10px] font-mono text-muted-foreground/60">
                {item.id}
              </span>
            </button>
          )
        })}
      </div>
    )
  },
)

MentionList.displayName = 'MentionList'
