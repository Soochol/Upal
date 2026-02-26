package services

import (
	"testing"
	"time"
)

func TestGenerationManagerLifecycle(t *testing.T) {
	gm := NewGenerationManager(time.Hour)
	defer gm.Stop()

	// Unknown ID returns not-found.
	if _, ok := gm.Get("nope"); ok {
		t.Fatal("expected not-found for unknown ID")
	}

	// Register → pending.
	gm.Register("g1")
	entry, ok := gm.Get("g1")
	if !ok {
		t.Fatal("expected entry after Register")
	}
	if entry.Status != GenerationPending {
		t.Fatalf("expected pending, got %q", entry.Status)
	}

	// Complete → completed with result.
	gm.Complete("g1", map[string]string{"name": "test-wf"})
	entry, _ = gm.Get("g1")
	if entry.Status != GenerationCompleted {
		t.Fatalf("expected completed, got %q", entry.Status)
	}
	if entry.Result == nil {
		t.Fatal("expected result after Complete")
	}

	// Register + Fail → failed with error message.
	gm.Register("g2")
	gm.Fail("g2", "llm timeout")
	entry, _ = gm.Get("g2")
	if entry.Status != GenerationFailed {
		t.Fatalf("expected failed, got %q", entry.Status)
	}
	if entry.Error != "llm timeout" {
		t.Fatalf("expected error message, got %q", entry.Error)
	}
}

func TestGenerationManagerGetReturnsSnapshot(t *testing.T) {
	gm := NewGenerationManager(time.Hour)
	defer gm.Stop()

	gm.Register("g1")
	snap, _ := gm.Get("g1")

	// Mutating the snapshot must not affect the entry in the manager.
	snap.Status = "tampered"
	entry, _ := gm.Get("g1")
	if entry.Status != GenerationPending {
		t.Fatalf("snapshot mutation leaked: got %q", entry.Status)
	}
}

func TestGenerationManagerGC(t *testing.T) {
	// TTL of 1ms so entries expire almost immediately.
	gm := &GenerationManager{
		entries: make(map[string]*GenerationEntry),
		ttl:     time.Millisecond,
		stop:    make(chan struct{}),
	}
	defer close(gm.stop)

	gm.Register("g1")
	gm.Complete("g1", "result")

	// Pending entries should not be collected.
	gm.Register("g2")

	time.Sleep(5 * time.Millisecond)
	gm.collectExpired()

	if _, ok := gm.Get("g1"); ok {
		t.Fatal("expected expired entry to be collected")
	}
	if _, ok := gm.Get("g2"); !ok {
		t.Fatal("pending entry should not be collected")
	}
}

func TestGenerationManagerCompleteUnknownID(t *testing.T) {
	gm := NewGenerationManager(time.Hour)
	defer gm.Stop()

	// Should not panic.
	gm.Complete("unknown", "result")
	gm.Fail("unknown", "err")
}
