import { registerNodeEditor } from '@/entities/node'
import { InputNodeEditor } from '../ui/InputNodeEditor'
import { AgentNodeEditor } from '../ui/AgentNodeEditor'
import { OutputNodeEditor } from '../ui/OutputNodeEditor'
import { ToolNodeEditor } from '../ui/ToolNodeEditor'
import { AssetNodeEditor } from '../ui/AssetNodeEditor'

export function registerAllEditors() {
  registerNodeEditor('input', InputNodeEditor)
  registerNodeEditor('agent', AgentNodeEditor)
  registerNodeEditor('output', OutputNodeEditor)
  registerNodeEditor('tool', ToolNodeEditor)
  registerNodeEditor('asset', AssetNodeEditor)
}
