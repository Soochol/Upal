import { cn } from "@/shared/lib/utils"
import type { ContentSessionStatus } from "@/shared/types"

const STATUS_CONFIG: Record<
  ContentSessionStatus,
  { label: string; className: string }
> = {
  draft:          { label: "Draft",         className: "bg-muted/15 text-muted-foreground border-border" },
  active:         { label: "Active",        className: "bg-success/15 text-success border-success/20" },
  collecting:     { label: "Collecting",    className: "bg-info/15 text-info border-info/20" },
  analyzing:      { label: "Analyzing",    className: "bg-info/15 text-info border-info/20" },
  pending_review: { label: "Pending",       className: "bg-warning/15 text-warning border-warning/20" },
  approved:       { label: "Approved",      className: "bg-success/15 text-success border-success/20" },
  rejected:       { label: "Rejected",      className: "bg-destructive/15 text-destructive border-destructive/20" },
  producing:      { label: "Producing",     className: "bg-primary/15 text-primary border-primary/20" },
  published:      { label: "Published",     className: "bg-success/20 text-success border-success/30" },
  error:          { label: "Error",        className: "bg-destructive/15 text-destructive border-destructive/20" },
}

type StatusBadgeProps = {
  status: ContentSessionStatus
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = STATUS_CONFIG[status] ?? { label: status, className: "bg-muted/15 text-muted-foreground border-border" }
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium",
        config.className,
        className,
      )}
    >
      {config.label}
    </span>
  )
}
