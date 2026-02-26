package services

import (
	"sync"
	"time"

	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
)

var _ ports.RunManagerPort = (*RunManager)(nil)

type runEntry struct {
	mu          sync.RWMutex
	events      []upal.EventRecord
	done        bool
	donePayload map[string]any
	subs        []chan struct{}
	completedAt time.Time
}

func (e *runEntry) snapshot(startSeq int) (events []upal.EventRecord, notify <-chan struct{}, done bool, donePayload map[string]any) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if startSeq < len(e.events) {
		events = make([]upal.EventRecord, len(e.events)-startSeq)
		copy(events, e.events[startSeq:])
	}

	ch := make(chan struct{})
	e.subs = append(e.subs, ch)

	return events, ch, e.done, e.donePayload
}

// RunManager tracks active workflow executions with per-run event buffering
// and subscriber fan-out for SSE streaming.
type RunManager struct {
	mu   sync.RWMutex
	runs map[string]*runEntry
	ttl  time.Duration
	stop chan struct{}
}

func NewRunManager(ttl time.Duration) *RunManager {
	rm := &RunManager{
		runs: make(map[string]*runEntry),
		ttl:  ttl,
		stop: make(chan struct{}),
	}
	go rm.gc()
	return rm
}

func (rm *RunManager) Stop() {
	close(rm.stop)
}

func (rm *RunManager) Register(runID string) {
	rm.mu.Lock()
	rm.runs[runID] = &runEntry{}
	rm.mu.Unlock()
}

func (rm *RunManager) Append(runID string, ev upal.EventRecord) {
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

	for _, ch := range subs {
		close(ch)
	}
}

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

func (rm *RunManager) Fail(runID string, errMsg string) {
	rm.Complete(runID, map[string]any{
		"status": "failed",
		"error":  errMsg,
	})
}

func (rm *RunManager) Subscribe(runID string, startSeq int) (events []upal.EventRecord, notify <-chan struct{}, done bool, donePayload map[string]any, found bool) {
	rm.mu.RLock()
	entry, ok := rm.runs[runID]
	rm.mu.RUnlock()
	if !ok {
		return nil, nil, false, nil, false
	}

	events, notify, done, donePayload = entry.snapshot(startSeq)
	return events, notify, done, donePayload, true
}

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
