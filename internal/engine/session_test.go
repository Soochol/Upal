package engine

import (
	"testing"

	"github.com/soochol/upal/internal/a2atypes"
)

func TestSessionManager_Create(t *testing.T) {
	mgr := NewSessionManager()
	sess := mgr.Create("wf-1")
	if sess.WorkflowID != "wf-1" {
		t.Errorf("workflow ID: got %q, want wf-1", sess.WorkflowID)
	}
	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}
	if sess.Status != SessionRunning {
		t.Errorf("status: got %q, want running", sess.Status)
	}
	if sess.State == nil {
		t.Error("state should be initialized")
	}
}

func TestSessionManager_Get(t *testing.T) {
	mgr := NewSessionManager()
	sess := mgr.Create("wf-1")
	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("session not found")
	}
	if got.ID != sess.ID {
		t.Errorf("ID: got %q, want %q", got.ID, sess.ID)
	}
}

func TestSessionManager_SetState(t *testing.T) {
	mgr := NewSessionManager()
	sess := mgr.Create("wf-1")
	mgr.SetState(sess.ID, "node1", "hello world")
	got, _ := mgr.Get(sess.ID)
	val, ok := got.State["node1"]
	if !ok {
		t.Fatal("state key 'node1' not found")
	}
	if val != "hello world" {
		t.Errorf("state value: got %q, want 'hello world'", val)
	}
}

func TestSessionManager_SetStatus(t *testing.T) {
	mgr := NewSessionManager()
	sess := mgr.Create("wf-1")
	mgr.SetStatus(sess.ID, SessionCompleted)
	got, _ := mgr.Get(sess.ID)
	if got.Status != SessionCompleted {
		t.Errorf("status: got %q, want completed", got.Status)
	}
}

func TestSessionManager_Artifacts(t *testing.T) {
	m := NewSessionManager()
	sess := m.Create("wf1")

	// Initially no artifacts
	arts := m.GetArtifacts(sess.ID, "node1")
	if len(arts) != 0 {
		t.Errorf("expected no artifacts, got %d", len(arts))
	}

	// Set artifacts
	m.SetArtifacts(sess.ID, "node1", []a2atypes.Artifact{{Name: "result", Parts: []a2atypes.Part{a2atypes.TextPart("hello")}}})
	arts = m.GetArtifacts(sess.ID, "node1")
	if len(arts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(arts))
	}

	// Get all artifacts
	all := m.GetAllArtifacts(sess.ID)
	if len(all) != 1 {
		t.Errorf("expected 1 node in artifacts, got %d", len(all))
	}

	// Set for another node
	m.SetArtifacts(sess.ID, "node2", []a2atypes.Artifact{{Name: "result2", Parts: []a2atypes.Part{a2atypes.TextPart("world")}}})
	all = m.GetAllArtifacts(sess.ID)
	if len(all) != 2 {
		t.Errorf("expected 2 nodes in artifacts, got %d", len(all))
	}

	// Non-existent session
	arts = m.GetArtifacts("nonexistent", "node1")
	if arts != nil {
		t.Errorf("expected nil for nonexistent session")
	}
	all = m.GetAllArtifacts("nonexistent")
	if all != nil {
		t.Errorf("expected nil for nonexistent session")
	}
}
