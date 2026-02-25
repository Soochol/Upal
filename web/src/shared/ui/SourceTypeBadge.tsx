import { cn } from "@/shared/lib/utils"
import type { SourceType } from "@/shared/types"

type SourceTypeBadgeProps = {
  type: SourceType
  className?: string
}

export function SourceTypeBadge({ type, className }: SourceTypeBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-medium",
        type === "static"
          ? "border-border text-muted-foreground bg-muted/30"
          : "border-primary/20 text-primary bg-primary/10",
        className,
      )}
    >
      {type === "static" ? "static" : "signal"}
    </span>
  )
}
