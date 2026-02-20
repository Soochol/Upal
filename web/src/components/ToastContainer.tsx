import { useUIStore, type Toast } from '@/stores/uiStore'
import { X, AlertCircle, CheckCircle2, Info } from 'lucide-react'
import { cn } from '@/lib/utils'

const variants: Record<Toast['variant'], { icon: typeof AlertCircle; className: string }> = {
  error: { icon: AlertCircle, className: 'border-destructive/30 bg-destructive/10 text-destructive' },
  success: { icon: CheckCircle2, className: 'border-green-500/30 bg-green-500/10 text-green-600 dark:text-green-400' },
  info: { icon: Info, className: 'border-border bg-card text-card-foreground' },
}

export function ToastContainer() {
  const toasts = useUIStore((s) => s.toasts)
  const dismissToast = useUIStore((s) => s.dismissToast)

  if (toasts.length === 0) return null

  return (
    <div className="fixed bottom-4 right-4 z-[100] flex flex-col gap-2 max-w-sm">
      {toasts.map((toast) => {
        const v = variants[toast.variant]
        const Icon = v.icon
        return (
          <div
            key={toast.id}
            className={cn(
              'flex items-start gap-2 rounded-lg border px-3 py-2.5 text-sm shadow-lg animate-in slide-in-from-right-5 fade-in duration-200',
              v.className,
            )}
          >
            <Icon className="h-4 w-4 mt-0.5 shrink-0" />
            <p className="flex-1 min-w-0 break-words">{toast.message}</p>
            <button
              onClick={() => dismissToast(toast.id)}
              className="shrink-0 opacity-60 hover:opacity-100 transition-opacity"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        )
      })}
    </div>
  )
}
