import { MainLayout } from '@/app/layout'
import { useSettingsStore } from '@/entities/settings/store'
import { useTheme } from '@/shared/ui/ThemeProvider'
import { Switch } from '@/shared/ui/switch'
import { Label } from '@/shared/ui/label'
import { Moon, Sun, Monitor } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

const THEME_OPTIONS = [
    { value: 'light' as const, label: 'Light', icon: Sun },
    { value: 'dark' as const, label: 'Dark', icon: Moon },
    { value: 'system' as const, label: 'System', icon: Monitor },
]

export default function SettingsPage() {
    const { showArchived, setShowArchived } = useSettingsStore()
    const { theme, setTheme } = useTheme()

    return (
        <MainLayout headerContent={<span className="font-semibold tracking-tight">Settings</span>}>
            <div className="flex-1 overflow-y-auto">
                <div className="max-w-2xl mx-auto px-6 py-10 space-y-10">

                    {/* Inbox */}
                    <section className="space-y-4">
                        <div>
                            <h2 className="text-lg font-semibold tracking-tight">Inbox</h2>
                            <p className="text-sm text-muted-foreground mt-0.5">인박스 표시 설정을 관리합니다.</p>
                        </div>
                        <div className="rounded-xl border border-border bg-card p-5">
                            <div className="flex items-center justify-between gap-4">
                                <div className="space-y-0.5">
                                    <Label htmlFor="show-archived" className="text-sm font-medium cursor-pointer">
                                        아카이브된 세션 표시
                                    </Label>
                                    <p className="text-xs text-muted-foreground">
                                        활성화하면 인박스에 아카이브된 세션도 함께 표시됩니다.
                                    </p>
                                </div>
                                <Switch
                                    id="show-archived"
                                    checked={showArchived}
                                    onCheckedChange={setShowArchived}
                                />
                            </div>
                        </div>
                    </section>

                    {/* Appearance */}
                    <section className="space-y-4">
                        <div>
                            <h2 className="text-lg font-semibold tracking-tight">Appearance</h2>
                            <p className="text-sm text-muted-foreground mt-0.5">테마와 외관을 설정합니다.</p>
                        </div>
                        <div className="rounded-xl border border-border bg-card p-5">
                            <div className="space-y-3">
                                <Label className="text-sm font-medium">Theme</Label>
                                <div className="flex gap-2">
                                    {THEME_OPTIONS.map((opt) => (
                                        <button
                                            key={opt.value}
                                            onClick={() => setTheme(opt.value)}
                                            className={cn(
                                                "flex items-center gap-2 px-4 py-2.5 rounded-lg border text-sm font-medium transition-all",
                                                theme === opt.value
                                                    ? "border-primary bg-primary/10 text-primary shadow-sm"
                                                    : "border-border text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                                            )}
                                        >
                                            <opt.icon className="size-4" />
                                            {opt.label}
                                        </button>
                                    ))}
                                </div>
                            </div>
                        </div>
                    </section>

                </div>
            </div>
        </MainLayout>
    )
}
