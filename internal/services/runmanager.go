package services

import (
	"sync"
	"time"
)

// EventRecord is a timestamped workflow event stored in the per-run buffer.
type EventRecord struct {
	Seq     int            `json:"seq"`
	Type    string         `json:"type"`
	NodeID  string         `json:"node_id,omitempty"`
	Payload map[string]any `json:"payload"`
}

// runEntry holds the in-memory state for a single run: buffered events,
// completion status, and subscriber notification channels.
type runEntry struct {
	mu          sync.RWMutex
	events      []EventRecord
	done        bool
	donePayload map[string]any // final "done" payload (status, session_id, state, run_id)
	subs        []chan struct{} // closed-and-replaced on each new event (fan-out wakeup)
	completedAt time.Time
}

// snapshot returns a copy of events from startSeq onward, registers a
// subscriber notification channel, and reports the run's done state.
func (e *runEntry) snapshot(startSeq int) (events []EventRecord, notify <-chan struct{}, done bool, donePayload map[string]any) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if startSeq < len(e.events) {
		events = make([]EventRecord, len(e.events)-startSeq)
		copy(events, e.events[startSeq:])
	}

	ch := make(chan struct{})
	e.subs = append(e.subs, ch)

	return events, ch, e.done, e.donePayload
}

// RunManager tracks in-progress and recently-completed workflow executions
// with an in-memory per-run event buffer and subscriber fan-out.
type RunManager struct {
	mu   sync.RWMutex
	runs map[string]*runEntry
	ttl  time.Duration
	stop chan struct{}
}

// NewRunManager creates a RunManager that keeps completed run buffers for
// the given TTL before garbage-collecting them.
func NewRunManager(ttl time.Duration) *RunManager {
	rm := &RunManager{
		runs: make(map[string]*runEntry),
		ttl:  ttl,
		stop: make(chan struct{}),
	}
	go rm.gc()
	return rm
}

// Stop terminates the GC goroutine.
func (rm *RunManager) Stop() {
	close(rm.stop)
}

// Register creates a new run entry. Call this when a run starts.
func (rm *RunManager) Register(runID string) {
	rm.mu.Lock()
	rm.runs[runID] = &runEntry{}
	rm.mu.Unlock()
}

// Append adds an event to the run's buffer and notifies all subscribers.
func (rm *RunManager) Append(runID string, ev EventRecord) {
	rm.mu.RLock()
	entry, ok := rm.runs[runID]
	rm.mu.RUnlock()
	if !ok {
		return
	}

	entry.mu.Lock()
	ev.Seq = len(entry.events)
	entry.events = append(entry.events, ev)
	subs := entry.subs
	entry.subs = nil
	entry.mu.Unlock()

	// Wake all subscribers by closing their channels.
	for _, ch := range subs {
		close(ch)
	}
}

// Complete marks a run as done with the given payload and notifies subscribers.
func (rm *RunManager) Complete(runID string, payload map[string]any) {
	rm.mu.RLock()
	entry, ok := rm.runs[runID]
	rm.mu.RUnlock()
	if !ok {
		return
	}

	entry.mu.Lock()
	entry.done = true
	entry.donePayload = payload
	entry.completedAt = time.Now()
	subs := entry.subs
	entry.subs = nil
	entry.mu.Unlock()

	for _, ch := range subs {
		close(ch)
	}
}

// Fail marks a run as done with an error and notifies subscribers.
func (rm *RunManager) Fail(runID string, errMsg string) {
	rm.Complete(runID, map[string]any{
		"status": "failed",
		"error":  errMsg,
	})
}

// Subscribe returns all buffered events from startSeq onward, a notification
// channel that is closed when new events arrive, and the run's done state.
// Returns found=false if the runID is not tracked.
func (rm *RunManager) Subscribe(runID string, startSeq int) (events []EventRecord, notify <-chan struct{}, done bool, donePayload map[string]any, found bool) {
	rm.mu.RLock()
	entry, ok := rm.runs[runID]
	rm.mu.RUnlock()
	if !ok {
		return nil, nil, false, nil, false
	}

	events, notify, done, donePayload = entry.snapshot(startSeq)
	return events, notify, done, donePayload, true
}

// Unsubscribe is a no-op cleanup hint. Subscriber channels are automatically
// removed when they are closed (on Append/Complete), so explicit cleanup is
// not strictly required. This method exists for symmetry with Subscribe.

// gc periodically removes completed run entries that have exceeded the TTL.
func (rm *RunManager) gc() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-rm.stop:
			return
		case <-ticker.C:
			rm.collectExpired()
		}
	}
}

func (rm *RunManager) collectExpired() {
	now := time.Now()
	rm.mu.Lock()
	defer rm.mu.Unlock()
	for id, entry := range rm.runs {
		entry.mu.RLock()
		expired := entry.done && now.Sub(entry.completedAt) > rm.ttl
		entry.mu.RUnlock()
		if expired {
			delete(rm.runs, id)
		}
	}
}
