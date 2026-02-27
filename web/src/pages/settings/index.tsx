import { useState, useEffect } from 'react'
import { MainLayout } from '@/app/layout'
import { useTheme } from '@/shared/ui/ThemeProvider'
import { Label } from '@/shared/ui/label'
import { Input } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/shared/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/select'
import { Moon, Sun, Monitor, Circle, CircleDot, Trash2, Plus } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { invalidateModelsCache } from '@/shared/api/useModels'
import {
    type AIProvider,
    type AIProviderCategory,
    ALL_CATEGORIES,
    CATEGORY_LABELS,
    PROVIDER_TYPES_BY_CATEGORY,
    MODELS_BY_PROVIDER_TYPE,
    listAIProviders,
    createAIProvider,
    updateAIProvider,
    deleteAIProvider,
    setAIProviderDefault,
} from '@/entities/ai-provider'

const THEME_OPTIONS = [
    { value: 'light' as const, label: 'Light', icon: Sun },
    { value: 'dark' as const, label: 'Dark', icon: Moon },
    { value: 'system' as const, label: 'System', icon: Monitor },
]

function getProviderTypeLabel(category: AIProviderCategory, type: string): string {
    const types = PROVIDER_TYPES_BY_CATEGORY[category]
    return types.find((t) => t.value === type)?.label ?? type
}

export default function SettingsPage() {
    const { theme, setTheme } = useTheme()
    const [providers, setProviders] = useState<AIProvider[]>([])
    const [addCategory, setAddCategory] = useState<AIProviderCategory | null>(null)
    const [addType, setAddType] = useState('')
    const [addModel, setAddModel] = useState('')
    const [addName, setAddName] = useState('')
    const [addApiKey, setAddApiKey] = useState('')
    const [saving, setSaving] = useState(false)
    const [error, setError] = useState('')

    const fetchProviders = () => {
        listAIProviders().then(setProviders).catch(() => setProviders([]))
    }

    useEffect(() => { fetchProviders() }, [])

    const handleCreate = async () => {
        if (!addCategory || !addType) return
        setSaving(true)
        setError('')
        try {
            const name = addName.trim() || getProviderTypeLabel(addCategory, addType)
            await createAIProvider({ name, category: addCategory, type: addType, model: addModel, api_key: addApiKey })
            invalidateModelsCache()
            fetchProviders()
            setAddCategory(null)
            setAddType('')
            setAddModel('')
            setAddName('')
            setAddApiKey('')
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save provider')
        } finally {
            setSaving(false)
        }
    }

    const handleDelete = async (provider: AIProvider) => {
        if (!confirm(`Delete provider "${provider.name}"?`)) return
        try {
            await deleteAIProvider(provider.id)
            invalidateModelsCache()
        } catch {
            // ignore — list refresh below shows current state
        } finally {
            fetchProviders()
        }
    }

    const handleSetDefault = async (id: string) => {
        try {
            const updated = await setAIProviderDefault(id)
            setProviders(updated)
            invalidateModelsCache()
        } catch {
            fetchProviders()
        }
    }

    const handleModelChange = async (provider: AIProvider, model: string) => {
        try {
            await updateAIProvider(provider.id, { model })
            invalidateModelsCache()
            fetchProviders()
        } catch {
            fetchProviders()
        }
    }

    const openAddDialog = (category: AIProviderCategory) => {
        setAddCategory(category)
        setAddType('')
        setAddModel('')
        setAddName('')
        setAddApiKey('')
        setError('')
    }

    return (
        <MainLayout headerContent={<span className="font-semibold tracking-tight">Settings</span>}>
            <div className="flex-1 overflow-y-auto">
                <div className="max-w-2xl mx-auto px-6 py-10 space-y-10">

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

                    {/* AI Providers */}
                    <section className="space-y-4">
                        <div>
                            <h2 className="text-lg font-semibold tracking-tight">AI Providers</h2>
                            <p className="text-sm text-muted-foreground mt-0.5">AI 제공자 API 키를 관리합니다.</p>
                        </div>

                        {ALL_CATEGORIES.map((category) => {
                            const categoryProviders = providers.filter((p) => p.category === category)
                            const typeOptions = PROVIDER_TYPES_BY_CATEGORY[category]

                            return (
                                <div key={category} className="space-y-2">
                                    <Label className="text-sm font-medium">{CATEGORY_LABELS[category]}</Label>
                                    <div className="rounded-xl border border-border bg-card">
                                        {categoryProviders.length > 0 && (
                                            <div className="divide-y divide-border">
                                                {categoryProviders.map((provider) => (
                                                    <div key={provider.id} className="flex items-center gap-3 px-5 py-3">
                                                        <button
                                                            onClick={() => handleSetDefault(provider.id)}
                                                            className={cn(
                                                                "shrink-0 transition-colors",
                                                                provider.is_default ? "text-primary" : "text-muted-foreground/40 hover:text-muted-foreground"
                                                            )}
                                                            title={provider.is_default ? 'Default provider' : 'Set as default'}
                                                        >
                                                            {provider.is_default
                                                                ? <CircleDot className="size-4" />
                                                                : <Circle className="size-4" />
                                                            }
                                                        </button>
                                                        <div className="flex-1 min-w-0 flex items-center gap-2">
                                                            <div className="min-w-0">
                                                                <span className="text-sm font-medium">{provider.name}</span>
                                                                <span className="text-xs text-muted-foreground ml-2">
                                                                    {getProviderTypeLabel(category, provider.type)}
                                                                </span>
                                                            </div>
                                                            {(MODELS_BY_PROVIDER_TYPE[provider.type]?.length ?? 0) > 0 && (
                                                                <Select
                                                                    value={provider.model}
                                                                    onValueChange={(v) => handleModelChange(provider, v)}
                                                                >
                                                                    <SelectTrigger className="h-7 w-auto min-w-[120px] max-w-[200px] text-xs gap-1 px-2">
                                                                        <SelectValue />
                                                                    </SelectTrigger>
                                                                    <SelectContent>
                                                                        {MODELS_BY_PROVIDER_TYPE[provider.type].map((m) => (
                                                                            <SelectItem key={m.value} value={m.value}>{m.label}</SelectItem>
                                                                        ))}
                                                                    </SelectContent>
                                                                </Select>
                                                            )}
                                                        </div>
                                                        <button
                                                            onClick={() => handleDelete(provider)}
                                                            className="text-muted-foreground hover:text-destructive transition-colors"
                                                            title="Delete provider"
                                                        >
                                                            <Trash2 className="size-4" />
                                                        </button>
                                                    </div>
                                                ))}
                                            </div>
                                        )}

                                        {typeOptions.length > 0 && (
                                            <div className={cn("px-5 py-3", categoryProviders.length > 0 && "border-t border-border")}>
                                                <button
                                                    onClick={() => openAddDialog(category)}
                                                    className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
                                                >
                                                    <Plus className="size-3.5" />
                                                    Add Provider
                                                </button>
                                            </div>
                                        )}

                                        {typeOptions.length === 0 && categoryProviders.length === 0 && (
                                            <div className="px-5 py-4 text-sm text-muted-foreground">
                                                No provider types available yet.
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )
                        })}
                    </section>

                </div>
            </div>

            {/* Add Provider Dialog */}
            <Dialog open={addCategory !== null} onOpenChange={(open) => { if (!open) setAddCategory(null) }}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>
                            Add {addCategory ? CATEGORY_LABELS[addCategory] : ''} Provider
                        </DialogTitle>
                    </DialogHeader>

                    <div className="space-y-4 py-2">
                        <div className="space-y-2">
                            <Label className="text-sm">Provider Type</Label>
                            <Select value={addType} onValueChange={(v) => {
                                setAddType(v)
                                const models = MODELS_BY_PROVIDER_TYPE[v] ?? []
                                setAddModel(models[0]?.value ?? '')
                            }}>
                                <SelectTrigger>
                                    <SelectValue placeholder="Select provider type..." />
                                </SelectTrigger>
                                <SelectContent>
                                    {addCategory && PROVIDER_TYPES_BY_CATEGORY[addCategory].map((t) => (
                                        <SelectItem key={t.value} value={t.value}>{t.label}</SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>

                        {addType && (MODELS_BY_PROVIDER_TYPE[addType]?.length ?? 0) > 0 && (
                            <div className="space-y-2">
                                <Label className="text-sm">Model</Label>
                                <Select value={addModel} onValueChange={setAddModel}>
                                    <SelectTrigger>
                                        <SelectValue placeholder="Select model..." />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {MODELS_BY_PROVIDER_TYPE[addType].map((m) => (
                                            <SelectItem key={m.value} value={m.value}>{m.label}</SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </div>
                        )}

                        <div className="space-y-2">
                            <Label className="text-sm">Name <span className="text-muted-foreground">(optional)</span></Label>
                            <Input
                                value={addName}
                                onChange={(e) => setAddName(e.target.value)}
                                placeholder={addType ? getProviderTypeLabel(addCategory!, addType) : 'Provider name'}
                            />
                        </div>

                        <div className="space-y-2">
                            <Label className="text-sm">API Key</Label>
                            <Input
                                type="password"
                                value={addApiKey}
                                onChange={(e) => setAddApiKey(e.target.value)}
                                placeholder="Enter API key"
                            />
                        </div>
                    </div>

                    {error && <p className="text-sm text-destructive">{error}</p>}

                    <DialogFooter>
                        <Button variant="outline" onClick={() => setAddCategory(null)}>Cancel</Button>
                        <Button onClick={handleCreate} disabled={!addType || saving}>
                            {saving ? 'Saving...' : 'Save'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </MainLayout>
    )
}
