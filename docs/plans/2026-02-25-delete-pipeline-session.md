# Pipeline / Session Delete Buttons Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add delete buttons to pipeline sidebar and session list, with confirmation dialogs and proper cleanup.

**Architecture:** Reuse existing Radix Dialog + Button components to build a `ConfirmDialog`. Add Trash2 icons to PipelineSidebar and SessionListPanel with hover-reveal pattern. Remove archive-required constraint from backend session delete. Wire mutations with query invalidation and URL cleanup.

**Tech Stack:** Go (backend service), React 19, TanStack Query, Radix Dialog, Tailwind CSS, lucide-react

---

### Task 1: Remove archive constraint from backend session delete

**Files:**
- Modify: `internal/services/content_session_service.go:437-460`
- Modify: `internal/services/content_session_service_test.go:168-192`

**Step 1: Update DeleteSession to remove archive guard**

In `internal/services/content_session_service.go`, remove the archive check (lines 438-444 become just the ID validation):

```go
func (s *ContentSessionService) DeleteSession(ctx context.Context, id string) error {
	if _, err := s.sessions.Get(ctx, id); err != nil {
		return err
	}

	// Clean up published_content (no FK cascade)
	if err := s.published.DeleteBySession(ctx, id); err != nil {
		return fmt.Errorf("delete published content: %w", err)
	}

	// Delete session (source_fetches + llm_analyses cascade in DB)
	if err := s.sessions.Delete(ctx, id); err != nil {
		return err
	}

	// Clean up workflow results
	_ = s.workflowResults.DeleteBySession(ctx, id)

	return nil
}
```

**Step 2: Update test to reflect removed constraint**

Replace `TestContentSessionService_DeleteRequiresArchived` with `TestContentSessionService_Delete`:

```go
func TestContentSessionService_Delete(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	svc.CreateSession(ctx, s)

	// Delete without archiving should succeed now
	if err := svc.DeleteSession(ctx, s.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	// Session should no longer exist
	_, err := svc.GetSession(ctx, s.ID)
	if err == nil {
		t.Error("expected error when getting deleted session")
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/services/... -v -race -run TestContentSessionService_Delete`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/services/content_session_service.go internal/services/content_session_service_test.go
git commit -m "feat: remove archive-required constraint from session delete"
```

---

### Task 2: Create ConfirmDialog shared component

**Files:**
- Create: `web/src/shared/ui/ConfirmDialog.tsx`

**Step 1: Create the component**

Build a reusable confirm dialog using existing `Dialog*` primitives and `Button`:

```tsx
import { useState } from 'react'
import { Loader2 } from 'lucide-react'
import {
  Dialog, DialogContent, DialogHeader, DialogFooter,
  DialogTitle, DialogDescription,
} from '@/shared/ui/dialog'
import { Button } from '@/shared/ui/button'

interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description: string
  confirmLabel?: string
  cancelLabel?: string
  destructive?: boolean
  isPending?: boolean
  onConfirm: () => void
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = 'Delete',
  cancelLabel = 'Cancel',
  destructive = true,
  isPending = false,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent showCloseButton={false} className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isPending}
          >
            {cancelLabel}
          </Button>
          <Button
            variant={destructive ? 'destructive' : 'default'}
            onClick={onConfirm}
            disabled={isPending}
          >
            {isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
```

**Step 2: Commit**

```bash
git add web/src/shared/ui/ConfirmDialog.tsx
git commit -m "feat: add reusable ConfirmDialog component"
```

---

### Task 3: Add delete button to PipelineSidebar

**Files:**
- Modify: `web/src/pages/pipelines/PipelineSidebar.tsx`
- Modify: `web/src/pages/Pipelines.tsx`

**Step 1: Add onDelete prop and delete UI to PipelineSidebar**

Add to imports: `Trash2` from lucide-react, `useMutation`/`useQueryClient` from tanstack, `deletePipeline` from API, `ConfirmDialog`.

Add `onDelete` callback prop to `PipelineSidebarProps`:
```ts
interface PipelineSidebarProps {
  pipelines: Pipeline[]
  selectedId: string | null
  onSelect: (id: string) => void
  isLoading: boolean
  onSettingsOpen?: () => void
  onDelete?: (id: string) => void  // NEW
}
```

Inside the component, add state and mutation:
```tsx
const queryClient = useQueryClient()
const [deleteTarget, setDeleteTarget] = useState<Pipeline | null>(null)

const deleteMutation = useMutation({
  mutationFn: (id: string) => deletePipeline(id),
  onSuccess: (_data, id) => {
    queryClient.invalidateQueries({ queryKey: ['pipelines'] })
    setDeleteTarget(null)
    onDelete?.(id)
  },
})
```

Add Trash2 button next to Settings button (inside the `<div className="flex items-center gap-1.5 shrink-0">` block):
```tsx
<button
  onClick={(e) => { e.stopPropagation(); setDeleteTarget(p) }}
  className="p-1 rounded-md text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
  title="Delete pipeline"
>
  <Trash2 className="h-3.5 w-3.5" />
</button>
```

Add ConfirmDialog at bottom of component return (before closing `</div>`):
```tsx
<ConfirmDialog
  open={!!deleteTarget}
  onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
  title="Delete pipeline"
  description={`"${deleteTarget?.name}" and all its sessions will be permanently deleted.`}
  isPending={deleteMutation.isPending}
  onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
/>
```

**Step 2: Wire onDelete callback in Pipelines.tsx**

Pass `onDelete` to `PipelineSidebar` that clears selected pipeline if it was deleted:

```tsx
<PipelineSidebar
  pipelines={pipelines}
  selectedId={selectedPipelineId}
  onSelect={selectPipeline}
  isLoading={isLoading}
  onDelete={(id) => {
    if (selectedPipelineId === id) setSearchParams({})
  }}
/>
```

**Step 3: Verify manually**

1. Hover a pipeline → Trash2 icon appears
2. Click → modal shows with pipeline name
3. Confirm → pipeline disappears, first pipeline auto-selected

**Step 4: Commit**

```bash
git add web/src/pages/pipelines/PipelineSidebar.tsx web/src/pages/Pipelines.tsx
git commit -m "feat: add pipeline delete button to sidebar"
```

---

### Task 4: Add delete button to SessionListPanel

**Files:**
- Modify: `web/src/pages/pipelines/SessionListPanel.tsx`
- Modify: `web/src/pages/Pipelines.tsx`

**Step 1: Add delete UI to SessionListPanel**

Add to imports: `Trash2` from lucide-react, `useMutation`/`useQueryClient` from tanstack, `deleteSession` from API, `ConfirmDialog`.

Add `onDelete` callback prop:
```ts
interface SessionListPanelProps {
  pipelineId: string
  selectedSessionId: string | null
  onSelectSession: (id: string) => void
  onNewSession?: () => void
  onDeleteSession?: (id: string) => void  // NEW
  className?: string
}
```

Inside the component, add state and mutation:
```tsx
const queryClient = useQueryClient()
const [deleteTarget, setDeleteTarget] = useState<typeof sessions[0] | null>(null)

const deleteMutation = useMutation({
  mutationFn: (id: string) => deleteSession(id),
  onSuccess: (_data, id) => {
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
    setDeleteTarget(null)
    onDeleteSession?.(id)
  },
})
```

Add Trash2 icon to each session item. Inside the `<div className="flex items-start justify-between gap-2 mb-1">`, add next to the date span:
```tsx
<button
  onClick={(e) => { e.stopPropagation(); setDeleteTarget(s) }}
  className="p-0.5 rounded-md text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer shrink-0"
  title="Delete session"
>
  <Trash2 className="h-3 w-3" />
</button>
```

Add ConfirmDialog at bottom of component return:
```tsx
<ConfirmDialog
  open={!!deleteTarget}
  onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
  title="Delete session"
  description={`"${deleteTarget?.name || `Session #${deleteTarget?.session_number}`}" will be permanently deleted.`}
  isPending={deleteMutation.isPending}
  onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
/>
```

**Step 2: Wire onDeleteSession in Pipelines.tsx**

Pass callback that clears selected session if it was deleted:

```tsx
<SessionListPanel
  pipelineId={selectedPipelineId}
  selectedSessionId={selectedSessionId}
  onSelectSession={selectSession}
  onNewSession={() => setShowNewSession(true)}
  onDeleteSession={(id) => {
    if (selectedSessionId === id) {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        next.delete('s')
        return next
      })
    }
  }}
  className={cn(...)}
/>
```

**Step 3: Verify manually**

1. Hover a session → Trash2 icon appears next to date
2. Click → modal shows with session name
3. Confirm → session disappears, first session auto-selected

**Step 4: Commit**

```bash
git add web/src/pages/pipelines/SessionListPanel.tsx web/src/pages/Pipelines.tsx
git commit -m "feat: add session delete button to session list"
```

---

### Task 5: Run full test suite and type-check

**Step 1: Backend tests**

Run: `make test`
Expected: All pass

**Step 2: Frontend type-check**

Run: `make test-frontend`
Expected: No type errors

**Step 3: Final commit if any fixups needed**

```bash
git add -A && git commit -m "fix: address type-check / test issues"
```
