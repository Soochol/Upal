package ports

import "github.com/soochol/upal/internal/upal"

// RunManagerPort defines the run event buffering and streaming boundary.
type RunManagerPort interface {
	Register(runID string)
	Append(runID string, ev upal.EventRecord)
	Complete(runID string, payload map[string]any)
	Fail(runID string, errMsg string)
	Subscribe(runID string, startSeq int) (events []upal.EventRecord, notify <-chan struct{}, done bool, donePayload map[string]any, found bool)
}

// ExecutionRegistryPort defines the execution pause/resume boundary.
type ExecutionRegistryPort interface {
	Register(runID string) *upal.ExecutionHandle
	Get(runID string) (*upal.ExecutionHandle, bool)
	Unregister(runID string)
}
