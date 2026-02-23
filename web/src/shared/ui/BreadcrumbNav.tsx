import { Link } from "react-router-dom"
import { ChevronRight } from "lucide-react"
import { cn } from "@/shared/lib/utils"

export type BreadcrumbItem = {
  label: string
  to?: string
}

type BreadcrumbNavProps = {
  items: BreadcrumbItem[]
  className?: string
}

export function BreadcrumbNav({ items, className }: BreadcrumbNavProps) {
  return (
    <nav className={cn("flex items-center gap-1 text-sm text-muted-foreground", className)}>
      {items.map((item, i) => (
        <span key={item.label} className="flex items-center gap-1">
          {i > 0 && <ChevronRight className="h-3.5 w-3.5 shrink-0" />}
          {item.to ? (
            <Link
              to={item.to}
              className="hover:text-foreground transition-colors"
            >
              {item.label}
            </Link>
          ) : (
            <span className="text-foreground font-medium">{item.label}</span>
          )}
        </span>
      ))}
    </nav>
  )
}
