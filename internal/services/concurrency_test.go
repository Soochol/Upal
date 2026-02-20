package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func TestConcurrencyLimiter_BasicAcquireRelease(t *testing.T) {
	limiter := NewConcurrencyLimiter(upal.ConcurrencyLimits{
		GlobalMax:   2,
		PerWorkflow: 1,
	})

	ctx := context.Background()

	// Acquire first slot.
	if err := limiter.Acquire(ctx, "wf-a"); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	stats := limiter.Stats()
	if stats.ActiveRuns != 1 {
		t.Fatalf("expected 1 active, got %d", stats.ActiveRuns)
	}

	// Release.
	limiter.Release("wf-a")
	stats = limiter.Stats()
	if stats.ActiveRuns != 0 {
		t.Fatalf("expected 0 active, got %d", stats.ActiveRuns)
	}
}

func TestConcurrencyLimiter_GlobalLimit(t *testing.T) {
	limiter := NewConcurrencyLimiter(upal.ConcurrencyLimits{
		GlobalMax:   2,
		PerWorkflow: 5,
	})

	ctx := context.Background()

	// Fill up global slots.
	limiter.Acquire(ctx, "wf-a")
	limiter.Acquire(ctx, "wf-b")

	// Third should block and timeout.
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	err := limiter.Acquire(timeoutCtx, "wf-c")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestConcurrencyLimiter_PerWorkflowLimit(t *testing.T) {
	limiter := NewConcurrencyLimiter(upal.ConcurrencyLimits{
		GlobalMax:   10,
		PerWorkflow: 1,
	})

	ctx := context.Background()

	// Fill per-workflow slot for wf-a.
	limiter.Acquire(ctx, "wf-a")

	// Second acquire for same workflow should block.
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	err := limiter.Acquire(timeoutCtx, "wf-a")
	if err == nil {
		t.Fatal("expected timeout error for per-workflow limit, got nil")
	}

	// Different workflow should still work.
	if err := limiter.Acquire(ctx, "wf-b"); err != nil {
		t.Fatalf("different workflow should succeed: %v", err)
	}

	limiter.Release("wf-a")
	limiter.Release("wf-b")
}

func TestConcurrencyLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewConcurrencyLimiter(upal.ConcurrencyLimits{
		GlobalMax:   5,
		PerWorkflow: 3,
	})

	ctx := context.Background()
	var wg sync.WaitGroup

	// Launch 10 goroutines, only 5 should run at a time (global limit).
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := limiter.Acquire(ctx, "test-wf"); err != nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
			limiter.Release("test-wf")
		}()
	}

	wg.Wait()

	stats := limiter.Stats()
	if stats.ActiveRuns != 0 {
		t.Fatalf("expected 0 active after all done, got %d", stats.ActiveRuns)
	}
}
