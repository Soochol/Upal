import { useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { fetchRun } from '@/entities/session-run'
import { runDisplayName, runPollingInterval } from '@/entities/session-run/constants'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import type { Run } from '@/entities/session-run'

interface InboxPreviewProps {
    runId: string
}

export function InboxPreview({ runId }: InboxPreviewProps) {
    const { data: run, isLoading } = useQuery({
        queryKey: ['session-run', runId],
        queryFn: () => fetchRun(runId),
        enabled: !!runId,
        refetchInterval: (query) => runPollingInterval(query.state.data as Run | undefined),
    })

    if (isLoading || !run) {
        return (
            <div className="flex-1 flex flex-col items-center justify-center p-6 text-muted-foreground gap-3">
                <Loader2 className="w-6 h-6 animate-spin text-primary/50" />
                <p className="text-sm font-medium">Loading run details...</p>
            </div>
        )
    }

    return (
        <div className="flex-1 h-full overflow-y-auto animate-in slide-in-from-right-4 duration-300">
            <div className="px-4 md:px-8 py-4 md:py-5 bg-background/80 backdrop-blur-sm z-10 flex flex-col gap-1.5">
                {run.session_name && (
                    <span className="text-xs font-bold uppercase tracking-widest text-primary/80 w-fit">
                        {run.session_name}
                    </span>
                )}
                <div className="flex items-center gap-3">
                    <h1 className="text-2xl font-bold tracking-tight text-foreground">
                        {run.name || runDisplayName(run)}
                    </h1>
                    <StatusBadge status={run.status} />
                </div>
                <div className="flex items-center gap-4 text-xs text-muted-foreground mt-1">
                    <span>{new Date(run.created_at).toLocaleString()}</span>
                    <span className="capitalize">{run.trigger_type}</span>
                    {run.source_count != null && run.source_count > 0 && <span>{run.source_count} sources</span>}
                </div>
            </div>

            <div className="max-w-4xl mx-auto px-6 py-6 space-y-6">
                {/* Analysis section */}
                {run.analysis && (
                    <section className="space-y-4">
                        <div>
                            <h3 className="text-sm font-semibold text-foreground mb-2">Analysis</h3>
                            <p className="text-sm text-muted-foreground">{run.analysis.summary}</p>
                        </div>

                        {/* Source highlights */}
                        {run.analysis.source_highlights && run.analysis.source_highlights.length > 0 && (
                            <div>
                                <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">Source Highlights</h4>
                                <div className="space-y-2">
                                    {run.analysis.source_highlights.map((sh, i) => (
                                        <div key={i} className="p-2.5 rounded-lg border border-border/50 bg-card/50">
                                            <span className="text-xs font-medium text-foreground">{sh.title}</span>
                                            <ul className="mt-1 space-y-0.5">
                                                {sh.key_points.map((point, j) => (
                                                    <li key={j} className="text-xs text-muted-foreground flex gap-2">
                                                        <span className="text-primary/60 shrink-0">&bull;</span>
                                                        {point}
                                                    </li>
                                                ))}
                                            </ul>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        {/* Cross-source insights */}
                        {run.analysis.insights && run.analysis.insights.length > 0 && (
                            <div>
                                <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">Insights</h4>
                                <ul className="space-y-1">
                                    {run.analysis.insights.map((insight, i) => (
                                        <li key={i} className="text-xs text-muted-foreground flex gap-2">
                                            <span className="text-primary/60">&bull;</span>
                                            {insight}
                                        </li>
                                    ))}
                                </ul>
                            </div>
                        )}
                    </section>
                )}

                {/* Sources section */}
                {run.sources && run.sources.length > 0 && (
                    <section>
                        <h3 className="text-sm font-semibold text-foreground mb-2">Sources ({run.sources.length})</h3>
                        <div className="space-y-3">
                            {run.sources.map((src) => (
                                <div key={src.id} className="p-3 rounded-lg border border-border/50 bg-card">
                                    <div className="flex items-center justify-between mb-2">
                                        <span className="text-sm font-medium">{src.label || src.tool}</span>
                                        <span className="text-xs text-muted-foreground">{src.count} items</span>
                                    </div>
                                    {src.items && src.items.length > 0 && (
                                        <ul className="space-y-1.5">
                                            {src.items.map((item, i) => (
                                                <li key={i} className="text-xs text-muted-foreground flex gap-2">
                                                    <span className="text-primary/60 shrink-0">&bull;</span>
                                                    {item.url ? (
                                                        <a href={item.url} target="_blank" rel="noopener noreferrer" className="hover:text-foreground hover:underline truncate">
                                                            {item.title}
                                                        </a>
                                                    ) : (
                                                        <span className="truncate">{item.title}</span>
                                                    )}
                                                </li>
                                            ))}
                                        </ul>
                                    )}
                                </div>
                            ))}
                        </div>
                    </section>
                )}

                {/* Workflow Runs section */}
                {run.workflow_runs && run.workflow_runs.length > 0 && (
                    <section>
                        <h3 className="text-sm font-semibold text-foreground mb-2">Workflow Runs</h3>
                        <div className="space-y-2">
                            {run.workflow_runs.map((wr, i) => (
                                <div key={i} className="p-3 rounded-lg border border-border/50 bg-card flex items-center justify-between">
                                    <div>
                                        <span className="text-sm font-medium">{wr.workflow_name}</span>
                                        {wr.error_message && (
                                            <p className="text-xs text-destructive mt-0.5">{wr.error_message}</p>
                                        )}
                                    </div>
                                    <StatusBadge status={wr.status} />
                                </div>
                            ))}
                        </div>
                    </section>
                )}
            </div>
        </div>
    )
}
