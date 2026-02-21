package upal

import (
	"fmt"
	"sync"
)

// ExecutionHandle represents a running workflow execution.
// It allows pipeline stages to pause and wait for an external signal via Resume.
type ExecutionHandle struct {
	RunID string

	mu      sync.Mutex
	waitChs map[string]chan map[string]any
}

// NewExecutionHandle creates a handle for a workflow run.
func NewExecutionHandle(runID string) *ExecutionHandle {
	return &ExecutionHandle{
		RunID:   runID,
		waitChs: make(map[string]chan map[string]any),
	}
}

// WaitForResume blocks until Resume is called for the given node.
func (h *ExecutionHandle) WaitForResume(nodeID string) map[string]any {
	h.mu.Lock()
	ch := make(chan map[string]any, 1)
	h.waitChs[nodeID] = ch
	h.mu.Unlock()
	return <-ch
}

// Resume unblocks a waiting node with the given payload.
func (h *ExecutionHandle) Resume(nodeID string, payload map[string]any) error {
	h.mu.Lock()
	ch, ok := h.waitChs[nodeID]
	if ok {
		delete(h.waitChs, nodeID)
	}
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("node %q is not waiting for resume", nodeID)
	}
	ch <- payload
	return nil
}
