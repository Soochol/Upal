package services

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
)

var _ ports.ConcurrencyControl = (*ConcurrencyLimiter)(nil)

// ConcurrencyLimiter controls how many workflows can execute simultaneously
// using channel-based semaphores at global and per-workflow levels.
type ConcurrencyLimiter struct {
	global      chan struct{}
	perWorkflow map[string]chan struct{}
	mu          sync.Mutex
	limits      upal.ConcurrencyLimits
	activeCount atomic.Int64
}

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

func (c *ConcurrencyLimiter) Acquire(ctx context.Context, workflowName string) error {
	select {
	case c.global <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	wfCh := c.getOrCreateWorkflowChan(workflowName)
	select {
	case wfCh <- struct{}{}:
		c.activeCount.Add(1)
		return nil
	case <-ctx.Done():
		<-c.global
		return ctx.Err()
	}
}

func (c *ConcurrencyLimiter) Release(workflowName string) {
	c.activeCount.Add(-1)

	c.mu.Lock()
	if ch, ok := c.perWorkflow[workflowName]; ok {
		select {
		case <-ch:
		default:
		}
	}
	c.mu.Unlock()

	select {
	case <-c.global:
	default:
	}
}

type ConcurrencyStats struct {
	ActiveRuns int `json:"active_runs"`
	GlobalMax  int `json:"global_max"`
	PerWorkflow int `json:"per_workflow"`
}

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
