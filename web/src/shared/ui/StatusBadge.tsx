import { cn } from "@/shared/lib/utils"
import type { ContentSessionStatus } from "@/shared/types"

const STATUS_CONFIG: Record<
  ContentSessionStatus,
  { label: string; className: string }
> = {
  collecting:     { label: "수집 중",   className: "bg-info/15 text-info border-info/20" },
  pending_review: { label: "리뷰 대기", className: "bg-warning/15 text-warning border-warning/20" },
  approved:       { label: "승인됨",    className: "bg-success/15 text-success border-success/20" },
  rejected:       { label: "거절됨",    className: "bg-destructive/15 text-destructive border-destructive/20" },
  producing:      { label: "제작 중",   className: "bg-primary/15 text-primary border-primary/20" },
  published:      { label: "게시 완료", className: "bg-success/20 text-success border-success/30" },
}

type StatusBadgeProps = {
  status: ContentSessionStatus
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = STATUS_CONFIG[status]
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
