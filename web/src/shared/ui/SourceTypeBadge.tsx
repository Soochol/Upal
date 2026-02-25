import { cn } from "@/shared/lib/utils"
import type { SourceType } from "@/shared/types"

type SourceTypeBadgeProps = {
  type: SourceType
  className?: string
}

const badgeStyles: Record<SourceType, string> = {
  static: "border-border text-muted-foreground bg-muted/30",
  signal: "border-primary/20 text-primary bg-primary/10",
  research: "border-[oklch(0.6_0.15_280)]/20 text-[oklch(0.6_0.15_280)] bg-[oklch(0.7_0.15_280)]/10",
}

export function SourceTypeBadge({ type, className }: SourceTypeBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-medium",
        badgeStyles[type],
        className,
      )}
    >
      {type}
    </span>
  )
}
