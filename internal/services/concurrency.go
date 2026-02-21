package services

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
)

var _ ports.ConcurrencyControl = (*ConcurrencyLimiter)(nil)

// ConcurrencyLimiter controls how many workflows can execute simultaneously.
// It uses channel-based counting semaphores at two levels: global and per-workflow.
type ConcurrencyLimiter struct {
	global      chan struct{}
	perWorkflow map[string]chan struct{}
	mu          sync.Mutex
	limits      upal.ConcurrencyLimits
	activeCount atomic.Int64
}

// NewConcurrencyLimiter creates a limiter with the given limits.
func NewConcurrencyLimiter(limits upal.ConcurrencyLimits) *ConcurrencyLimiter {
	if limits.GlobalMax <= 0 {
		limits.GlobalMax = 10
	}
	if limits.PerWorkflow <= 0 {
		limits.PerWorkflow = 3
	}

	return &ConcurrencyLimiter{
		global:      make(chan struct{}, limits.GlobalMax),
		perWorkflow: make(map[string]chan struct{}),
		limits:      limits,
	}
}

// Acquire blocks until both global and per-workflow slots are available,
// or returns an error if the context is cancelled.
func (c *ConcurrencyLimiter) Acquire(ctx context.Context, workflowName string) error {
	// 1. Acquire global slot.
	select {
	case c.global <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	// 2. Acquire per-workflow slot.
	wfCh := c.getOrCreateWorkflowChan(workflowName)
	select {
	case wfCh <- struct{}{}:
		c.activeCount.Add(1)
		return nil
	case <-ctx.Done():
		// Release global slot since we couldn't get per-workflow.
		<-c.global
		return ctx.Err()
	}
}

// Release returns both the global and per-workflow slots.
func (c *ConcurrencyLimiter) Release(workflowName string) {
	c.activeCount.Add(-1)

	// Release per-workflow slot.
	c.mu.Lock()
	if ch, ok := c.perWorkflow[workflowName]; ok {
		select {
		case <-ch:
		default:
		}
	}
	c.mu.Unlock()

	// Release global slot.
	select {
	case <-c.global:
	default:
	}
}

// ConcurrencyStats reports current usage.
type ConcurrencyStats struct {
	ActiveRuns int `json:"active_runs"`
	GlobalMax  int `json:"global_max"`
	PerWorkflow int `json:"per_workflow"`
}

// Stats returns the current concurrency statistics.
func (c *ConcurrencyLimiter) Stats() ConcurrencyStats {
	return ConcurrencyStats{
		ActiveRuns:  int(c.activeCount.Load()),
		GlobalMax:   c.limits.GlobalMax,
		PerWorkflow: c.limits.PerWorkflow,
	}
}

func (c *ConcurrencyLimiter) getOrCreateWorkflowChan(name string) chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch, ok := c.perWorkflow[name]
	if !ok {
		ch = make(chan struct{}, c.limits.PerWorkflow)
		c.perWorkflow[name] = ch
	}
	return ch
}
