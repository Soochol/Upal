package services

import (
	"sync"
	"time"
)

// GenerationManager tracks background LLM generation jobs (workflow, pipeline)
// with in-memory result storage and TTL-based cleanup.
type GenerationManager struct {
	mu      sync.RWMutex
	entries map[string]*GenerationEntry
	ttl     time.Duration
	stop    chan struct{}
}

// Generation lifecycle states.
const (
	GenerationPending   = "pending"
	GenerationCompleted = "completed"
	GenerationFailed    = "failed"
)

// GenerationEntry represents a single generation job.
type GenerationEntry struct {
	Status      string `json:"status"` // GenerationPending, GenerationCompleted, GenerationFailed
	Result      any    `json:"result,omitempty"`
	Error       string `json:"error,omitempty"`
	createdAt   time.Time
	completedAt time.Time
}

// pendingTimeout is the maximum time a generation can stay in pending state
// before GC cleans it up (e.g. if the goroutine panics or hangs).
const pendingTimeout = 10 * time.Minute

func NewGenerationManager(ttl time.Duration) *GenerationManager {
	gm := &GenerationManager{
		entries: make(map[string]*GenerationEntry),
		ttl:     ttl,
		stop:    make(chan struct{}),
	}
	go gm.gc()
	return gm
}

func (gm *GenerationManager) Stop() {
	close(gm.stop)
}

// Register creates a new pending generation entry.
func (gm *GenerationManager) Register(id string) {
	gm.mu.Lock()
	gm.entries[id] = &GenerationEntry{Status: GenerationPending, createdAt: time.Now()}
	gm.mu.Unlock()
}

// Complete marks a generation as completed with the given result.
func (gm *GenerationManager) Complete(id string, result any) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	entry, ok := gm.entries[id]
	if !ok {
		return
	}
	entry.Status = GenerationCompleted
	entry.Result = result
	entry.completedAt = time.Now()
}

// Fail marks a generation as failed with an error message.
func (gm *GenerationManager) Fail(id string, errMsg string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	entry, ok := gm.entries[id]
	if !ok {
		return
	}
	entry.Status = GenerationFailed
	entry.Error = errMsg
	entry.completedAt = time.Now()
}

// Get returns a snapshot of a generation entry.
func (gm *GenerationManager) Get(id string) (*GenerationEntry, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	entry, ok := gm.entries[id]
	if !ok {
		return nil, false
	}
	cp := *entry
	return &cp, true
}

func (gm *GenerationManager) gc() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-gm.stop:
			return
		case <-ticker.C:
			gm.collectExpired()
		}
	}
}

func (gm *GenerationManager) collectExpired() {
	now := time.Now()
	gm.mu.Lock()
	defer gm.mu.Unlock()
	for id, entry := range gm.entries {
		switch {
		case entry.Status == GenerationPending && now.Sub(entry.createdAt) > pendingTimeout:
			// Pending too long — goroutine likely panicked or hung.
			delete(gm.entries, id)
		case entry.Status != GenerationPending && now.Sub(entry.completedAt) > gm.ttl:
			delete(gm.entries, id)
		}
	}
}
