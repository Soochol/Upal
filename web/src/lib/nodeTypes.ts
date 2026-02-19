import { Inbox, Bot, Wrench, ArrowRightFromLine, Globe } from 'lucide-react'

export const nodeIconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  input: Inbox,
  agent: Bot,
  tool: Wrench,
  output: ArrowRightFromLine,
  external: Globe,
}
