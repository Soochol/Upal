package engine

import "testing"

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
