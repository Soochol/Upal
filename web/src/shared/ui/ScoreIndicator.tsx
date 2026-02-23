import { cn } from "@/shared/lib/utils"

type ScoreIndicatorProps = {
  score: number   // 0–100
  className?: string
}

function scoreColor(score: number) {
  if (score >= 85) return "text-success"
  if (score >= 65) return "text-warning"
  return "text-muted-foreground"
}

export function ScoreIndicator({ score, className }: ScoreIndicatorProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 text-xs font-semibold tabular-nums",
        scoreColor(score),
        className,
      )}
    >
      {score}
    </span>
  )
}
