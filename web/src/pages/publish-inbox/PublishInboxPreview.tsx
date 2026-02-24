import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchContentSession, publishSession } from '@/entities/content-session/api'
import { fetchPublishChannels } from '@/entities/publish-channel/api'
import { apiFetch } from '@/shared/api/client'
import type { WorkflowResult } from '@/entities/content-session/types'
import type { PublishChannel } from '@/entities/publish-channel/types'
import { Loader2, CheckCircle2, XCircle, Send, ExternalLink, Clock } from 'lucide-react'

async function rejectWorkflowResult(sessionId: string, runId: string) {
    return apiFetch(`/api/content-sessions/${encodeURIComponent(sessionId)}/reject-result`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ run_id: runId }),
    })
}

type Props = { sessionId: string }

export function PublishInboxPreview({ sessionId }: Props) {
    const queryClient = useQueryClient()
    const [publishingRunId, setPublishingRunId] = useState<string | null>(null)
    const [rejectingRunId, setRejectingRunId] = useState<string | null>(null)

    const { data: session, isLoading } = useQuery({
        queryKey: ['content-session', sessionId],
        queryFn: () => fetchContentSession(sessionId),
        refetchInterval: 3000,
    })

    const { data: channels = [] } = useQuery({
        queryKey: ['publish-channels'],
        queryFn: fetchPublishChannels,
    })

    const channelMap = Object.fromEntries(channels.map(ch => [ch.id, ch]))

    if (isLoading || !session) {
        return (
            <div className="flex-1 flex items-center justify-center">
                <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
        )
    }

    const results = session.workflow_results ?? []
    const actionableResults = results.filter(r => r.status === 'success')
    const terminalResults = results.filter(r => r.status === 'published' || r.status === 'rejected' || r.status === 'failed')

    const handlePublish = async (runId: string) => {
        setPublishingRunId(runId)
        try {
            await publishSession(sessionId, [runId])
            queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
            queryClient.invalidateQueries({ queryKey: ['publish-inbox-sessions'] })
        } finally {
            setPublishingRunId(null)
        }
    }

    const handleReject = async (runId: string) => {
        setRejectingRunId(runId)
        try {
            await rejectWorkflowResult(sessionId, runId)
            queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
            queryClient.invalidateQueries({ queryKey: ['publish-inbox-sessions'] })
        } finally {
            setRejectingRunId(null)
        }
    }

    return (
        <div className="flex-1 overflow-y-auto p-6">
            <div className="max-w-4xl mx-auto">
                {/* Header */}
                <div className="mb-6">
                    <h2 className="text-lg font-bold tracking-tight">
                        {session.pipeline_name || 'Pipeline'} — Session #{session.session_number || session.id.slice(0, 8)}
                    </h2>
                    <p className="text-sm text-muted-foreground mt-1">
                        {actionableResults.length} workflow{actionableResults.length !== 1 ? 's' : ''} ready for review
                        {terminalResults.length > 0 && ` · ${terminalResults.length} processed`}
                    </p>
                </div>

                {/* Workflow cards */}
                <div className="space-y-4">
                    {results.map((wr) => (
                        <WorkflowResultCard
                            key={wr.run_id || wr.workflow_name}
                            result={wr}
                            channel={wr.channel_id ? channelMap[wr.channel_id] : undefined}
                            isPublishing={publishingRunId === wr.run_id}
                            isRejecting={rejectingRunId === wr.run_id}
                            onPublish={() => handlePublish(wr.run_id)}
                            onReject={() => handleReject(wr.run_id)}
                        />
                    ))}
                </div>
            </div>
        </div>
    )
}

function WorkflowResultCard({ result, channel, isPublishing, isRejecting, onPublish, onReject }: {
    result: WorkflowResult
    channel?: PublishChannel
    isPublishing: boolean
    isRejecting: boolean
    onPublish: () => void
    onReject: () => void
}) {
    const isTerminal = result.status === 'published' || result.status === 'rejected' || result.status === 'failed'
    const isActionable = result.status === 'success'

    const statusConfig: Record<string, { bg: string; text: string; icon: typeof CheckCircle2; label: string }> = {
        success: { bg: 'border-info/30 bg-info/5', text: 'text-info', icon: Clock, label: 'Awaiting Review' },
        published: { bg: 'border-success/30 bg-success/5', text: 'text-success', icon: CheckCircle2, label: 'Published' },
        rejected: { bg: 'border-muted bg-muted/10', text: 'text-muted-foreground', icon: XCircle, label: 'Rejected' },
        failed: { bg: 'border-destructive/30 bg-destructive/5', text: 'text-destructive', icon: XCircle, label: 'Failed' },
        running: { bg: 'border-info/50 bg-info/5', text: 'text-info', icon: Loader2, label: 'Running' },
        pending: { bg: 'border-border bg-muted/5', text: 'text-muted-foreground', icon: Clock, label: 'Pending' },
    }

    const config = statusConfig[result.status] || statusConfig.pending
    const StatusIcon = config.icon

    return (
        <div className={`rounded-xl border ${config.bg} transition-all ${isTerminal ? 'opacity-60' : ''}`}>
            {/* Header */}
            <div className="flex items-center justify-between px-5 py-3 border-b border-border/30">
                <div className="flex items-center gap-3">
                    <StatusIcon className={`h-4 w-4 ${config.text} ${result.status === 'running' ? 'animate-spin' : ''}`} />
                    <span className="text-sm font-semibold">{result.workflow_name}</span>
                </div>
                <div className="flex items-center gap-2">
                    {channel && (
                        <span className="text-xs font-medium bg-muted/50 px-2.5 py-1 rounded-full text-muted-foreground">
                            {channel.name} ({channel.type})
                        </span>
                    )}
                    <span className={`text-xs font-medium px-2.5 py-1 rounded-full ${config.text} bg-current/10`}>
                        {config.label}
                    </span>
                </div>
            </div>

            {/* Content area */}
            <div className="px-5 py-4">
                {result.run_id && (
                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                        <span>Run: {result.run_id.slice(0, 12)}</span>
                        {result.completed_at && (
                            <span>Completed: {new Date(result.completed_at).toLocaleString()}</span>
                        )}
                        {result.output_url && (
                            <a href={result.output_url} target="_blank" rel="noopener noreferrer"
                                className="flex items-center gap-1 text-primary hover:underline">
                                <ExternalLink className="h-3 w-3" /> Preview
                            </a>
                        )}
                    </div>
                )}
            </div>

            {/* Actions */}
            {isActionable && (
                <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-border/30">
                    <button
                        onClick={onReject}
                        disabled={isRejecting}
                        className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-medium
                            text-muted-foreground hover:text-destructive hover:bg-destructive/10
                            transition-colors cursor-pointer disabled:opacity-50"
                    >
                        {isRejecting ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <XCircle className="h-3.5 w-3.5" />}
                        Reject
                    </button>
                    <button
                        onClick={onPublish}
                        disabled={isPublishing}
                        className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-medium
                            bg-success text-success-foreground hover:bg-success/90
                            transition-colors cursor-pointer disabled:opacity-50 shadow-sm"
                    >
                        {isPublishing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Send className="h-3.5 w-3.5" />}
                        Publish
                    </button>
                </div>
            )}
        </div>
    )
}
