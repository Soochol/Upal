import { Inbox, Bot, ArrowRightFromLine, GitBranch, Repeat, Bell, Radar, UserCheck, Workflow } from 'lucide-react'
import type { ComponentType } from 'react'

export type NodeType = 'input' | 'agent' | 'output' | 'branch' | 'iterator' | 'notification' | 'sensor' | 'approval' | 'subworkflow'

export type NodeTypeConfig = {
  type: NodeType
  label: string
  description: string
  icon: ComponentType<{ className?: string }>
  border: string
  borderSelected: string
  headerBg: string
  accent: string
  glow: string
  paletteBg: string
  cssVar: string
}

export const NODE_TYPES: Record<NodeType, NodeTypeConfig> = {
  input: {
    type: 'input',
    label: 'User Input',
    description: 'User-provided data entry point',
    icon: Inbox,
    border: 'border-node-input/30',
    borderSelected: 'border-node-input',
    headerBg: 'bg-node-input/15',
    accent: 'bg-node-input text-node-input-foreground',
    glow: 'shadow-[0_0_16px_oklch(0.795_0.184_86.047/0.4)]',
    paletteBg: 'bg-node-input/15 text-node-input border-node-input/30 hover:bg-node-input/25',
    cssVar: 'var(--node-input)',
  },
  agent: {
    type: 'agent',
    label: 'Agent',
    description: 'AI model processing step',
    icon: Bot,
    border: 'border-node-agent/30',
    borderSelected: 'border-node-agent',
    headerBg: 'bg-node-agent/15',
    accent: 'bg-node-agent text-node-agent-foreground',
    glow: 'shadow-[0_0_16px_oklch(0.588_0.158_241.966/0.4)]',
    paletteBg: 'bg-node-agent/15 text-node-agent border-node-agent/30 hover:bg-node-agent/25',
    cssVar: 'var(--node-agent)',
  },
  output: {
    type: 'output',
    label: 'Output',
    description: 'Workflow result endpoint',
    icon: ArrowRightFromLine,
    border: 'border-node-output/30',
    borderSelected: 'border-node-output',
    headerBg: 'bg-node-output/15',
    accent: 'bg-node-output text-node-output-foreground',
    glow: 'shadow-[0_0_16px_oklch(0.648_0.2_142.495/0.4)]',
    paletteBg: 'bg-node-output/15 text-node-output border-node-output/30 hover:bg-node-output/25',
    cssVar: 'var(--node-output)',
  },
  branch: {
    type: 'branch',
    label: 'Branch',
    description: 'Conditional routing based on expression or LLM',
    icon: GitBranch,
    border: 'border-node-branch/30',
    borderSelected: 'border-node-branch',
    headerBg: 'bg-node-branch/15',
    accent: 'bg-node-branch text-node-branch-foreground',
    glow: 'shadow-[0_0_16px_oklch(0.627_0.195_303.9/0.4)]',
    paletteBg: 'bg-node-branch/15 text-node-branch border-node-branch/30 hover:bg-node-branch/25',
    cssVar: 'var(--node-branch)',
  },
  iterator: {
    type: 'iterator',
    label: 'Iterator',
    description: 'Loop over array items',
    icon: Repeat,
    border: 'border-node-iterator/30',
    borderSelected: 'border-node-iterator',
    headerBg: 'bg-node-iterator/15',
    accent: 'bg-node-iterator text-node-iterator-foreground',
    glow: 'shadow-[0_0_16px_oklch(0.65_0.17_195/0.4)]',
    paletteBg: 'bg-node-iterator/15 text-node-iterator border-node-iterator/30 hover:bg-node-iterator/25',
    cssVar: 'var(--node-iterator)',
  },
  notification: {
    type: 'notification',
    label: 'Notification',
    description: 'Send message to external channel',
    icon: Bell,
    border: 'border-node-notification/30',
    borderSelected: 'border-node-notification',
    headerBg: 'bg-node-notification/15',
    accent: 'bg-node-notification text-node-notification-foreground',
    glow: 'shadow-[0_0_16px_oklch(0.7_0.18_55/0.4)]',
    paletteBg: 'bg-node-notification/15 text-node-notification border-node-notification/30 hover:bg-node-notification/25',
    cssVar: 'var(--node-notification)',
  },
  sensor: {
    type: 'sensor',
    label: 'Sensor',
    description: 'Wait for external condition or webhook',
    icon: Radar,
    border: 'border-node-sensor/30',
    borderSelected: 'border-node-sensor',
    headerBg: 'bg-node-sensor/15',
    accent: 'bg-node-sensor text-node-sensor-foreground',
    glow: 'shadow-[0_0_16px_oklch(0.6_0.16_200/0.4)]',
    paletteBg: 'bg-node-sensor/15 text-node-sensor border-node-sensor/30 hover:bg-node-sensor/25',
    cssVar: 'var(--node-sensor)',
  },
  approval: {
    type: 'approval',
    label: 'Approval',
    description: 'Request human approval before proceeding',
    icon: UserCheck,
    border: 'border-node-approval/30',
    borderSelected: 'border-node-approval',
    headerBg: 'bg-node-approval/15',
    accent: 'bg-node-approval text-node-approval-foreground',
    glow: 'shadow-[0_0_16px_oklch(0.55_0.2_275/0.4)]',
    paletteBg: 'bg-node-approval/15 text-node-approval border-node-approval/30 hover:bg-node-approval/25',
    cssVar: 'var(--node-approval)',
  },
  subworkflow: {
    type: 'subworkflow',
    label: 'Sub-Workflow',
    description: 'Execute another workflow as a child',
    icon: Workflow,
    border: 'border-node-subworkflow/30',
    borderSelected: 'border-node-subworkflow',
    headerBg: 'bg-node-subworkflow/15',
    accent: 'bg-node-subworkflow text-node-subworkflow-foreground',
    glow: 'shadow-[0_0_16px_oklch(0.5_0.05_250/0.4)]',
    paletteBg: 'bg-node-subworkflow/15 text-node-subworkflow border-node-subworkflow/30 hover:bg-node-subworkflow/25',
    cssVar: 'var(--node-subworkflow)',
  },
}

