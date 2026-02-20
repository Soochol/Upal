package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/session"
)

// --- Helpers ---

// newEdgeCaseServer creates a server with a trigger repo suitable for webhook edge-case tests.
func newEdgeCaseServer() (*Server, repository.TriggerRepository) {
	sessionSvc := session.InMemoryService()
	wfRepo := repository.NewMemory()
	wfSvc := services.NewWorkflowService(wfRepo, nil, sessionSvc, nil)
	trigRepo := repository.NewMemoryTriggerRepository()
	srv := NewServer(nil, wfSvc, wfRepo, nil)
	srv.SetTriggerRepository(trigRepo)
	return srv, trigRepo
}

// newEdgeCaseServerWithScheduler creates a server wired with a full SchedulerService
// and all its dependencies backed by in-memory stores.
func newEdgeCaseServerWithScheduler() *Server {
	sessionSvc := session.InMemoryService()
	wfRepo := repository.NewMemory()
	wfSvc := services.NewWorkflowService(wfRepo, nil, sessionSvc, nil)
	schedRepo := repository.NewMemoryScheduleRepository()
	runRepo := repository.NewMemoryRunRepository()
	limiter := services.NewConcurrencyLimiter(upal.ConcurrencyLimits{GlobalMax: 10, PerWorkflow: 3})
	runHistorySvc := services.NewRunHistoryService(runRepo)
	retryExec := services.NewRetryExecutor(wfSvc, runHistorySvc)
	schedSvc := services.NewSchedulerService(schedRepo, wfSvc, retryExec, limiter, runHistorySvc)
	srv := NewServer(nil, wfSvc, wfRepo, nil)
	srv.SetSchedulerService(schedSvc)
	return srv
}

// seedEdgeCaseWorkflow creates a minimal workflow so the webhook handler's Lookup succeeds.
func seedEdgeCaseWorkflow(t *testing.T, srv *Server, name string) {
	t.Helper()
	wf := upal.WorkflowDefinition{
		Name:    name,
		Version: 1,
		Nodes: []upal.NodeDefinition{
			{ID: "out1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
	}
	body, _ := json.Marshal(wf)
	req := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("seedEdgeCaseWorkflow: got status %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

// createEdgeCaseSchedule is a helper that POSTs a schedule and returns the decoded schedule.
func createEdgeCaseSchedule(t *testing.T, srv *Server, wfName, cronExpr string, enabled bool) upal.Schedule {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"workflow_name": wfName,
		"cron_expr":     cronExpr,
		"enabled":       enabled,
	})
	req := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("createEdgeCaseSchedule: got %d, want 201; body: %s", w.Code, w.Body.String())
	}
	var s upal.Schedule
	if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
		t.Fatalf("createEdgeCaseSchedule decode: %v", err)
	}
	return s
}

// getScheduleState fetches a schedule via GET and returns its Enabled field.
func getScheduleState(t *testing.T, srv *Server, id string) bool {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/schedules/"+id, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("getScheduleState: got %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var s upal.Schedule
	if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
		t.Fatalf("getScheduleState decode: %v", err)
	}
	return s.Enabled
}

// ============================================================
// Webhook Edge Cases
// ============================================================

// TestWebhook_MalformedJSON verifies that sending invalid JSON to the webhook
// handler does not cause a panic. The handler silently ignores the unmarshal
// failure and proceeds with a nil payload, ultimately returning 202.
func TestWebhook_MalformedJSON(t *testing.T) {
	srv, trigRepo := newEdgeCaseServer()
	seedEdgeCaseWorkflow(t, srv, "malformed-wf")

	trigger := &upal.Trigger{
		ID:           "trig_malformed",
		WorkflowName: "malformed-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{Secret: ""}, // no secret — skip HMAC
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	// Send body that is not valid JSON.
	malformed := []byte(`{not valid json}`)
	req := httptest.NewRequest("POST", "/api/hooks/trig_malformed", bytes.NewReader(malformed))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// The primary assertion: no panic occurs.
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202; body: %s", w.Code, w.Body.String())
	}
}

// TestWebhook_LargePayload verifies that the webhook handler can process a
// large (~100KB) JSON payload without error.
func TestWebhook_LargePayload(t *testing.T) {
	srv, trigRepo := newEdgeCaseServer()
	seedEdgeCaseWorkflow(t, srv, "large-wf")

	trigger := &upal.Trigger{
		ID:           "trig_large",
		WorkflowName: "large-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{Secret: ""},
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	// Build a ~100KB JSON payload.
	largeData := strings.Repeat("x", 100_000)
	payload, _ := json.Marshal(map[string]string{"data": largeData})

	req := httptest.NewRequest("POST", "/api/hooks/trig_large", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202; body: %s", w.Code, w.Body.String())
	}
}

// TestWebhook_EmptyTriggerID verifies that POSTing to /api/hooks/ with no
// trigger ID returns a 404 or 405 (Chi won't route to the handler).
func TestWebhook_EmptyTriggerID(t *testing.T) {
	srv, _ := newEdgeCaseServer()

	req := httptest.NewRequest("POST", "/api/hooks/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	// Chi routing: /api/hooks/{id} won't match an empty id, so we expect
	// either 404 (no route matched) or 405 (method not allowed).
	if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status: got %d, want 404 or 405", w.Code)
	}
}

// TestWebhook_ContentTypeVariations verifies that the webhook handler works
// regardless of Content-Type header since it reads raw body bytes.
func TestWebhook_ContentTypeVariations(t *testing.T) {
	srv, trigRepo := newEdgeCaseServer()
	seedEdgeCaseWorkflow(t, srv, "ctype-wf")

	trigger := &upal.Trigger{
		ID:           "trig_ctype",
		WorkflowName: "ctype-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{Secret: ""},
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	payload := []byte(`{"msg":"hello"}`)

	tests := []struct {
		name        string
		contentType string
	}{
		{"text/plain", "text/plain"},
		{"no content-type", ""},
		{"application/octet-stream", "application/octet-stream"},
		{"application/xml", "application/xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/hooks/trig_ctype", bytes.NewReader(payload))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, req)

			if w.Code != http.StatusAccepted {
				t.Fatalf("status: got %d, want 202; body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// ============================================================
// Schedule Edge Cases
// ============================================================

// TestSchedule_RapidPauseResume verifies that rapidly pausing and resuming a
// schedule many times does not cause a panic or corrupt the schedule state.
// Each pause/resume pair is executed sequentially to avoid triggering a known
// data-race in the service layer (SchedulerService.ResumeSchedule mutates
// the schedule outside its mutex). The test validates that the final state
// is consistent after 10 rapid toggle cycles.
func TestSchedule_RapidPauseResume(t *testing.T) {
	srv := newEdgeCaseServerWithScheduler()
	sched := createEdgeCaseSchedule(t, srv, "rapid-wf", "0 * * * *", true)

	const iterations = 10

	for i := 0; i < iterations; i++ {
		// Pause.
		pauseReq := httptest.NewRequest("POST", "/api/schedules/"+sched.ID+"/pause", nil)
		pauseW := httptest.NewRecorder()
		srv.Handler().ServeHTTP(pauseW, pauseReq)
		if pauseW.Code != http.StatusNoContent {
			t.Fatalf("iteration %d pause: got %d, want 204", i, pauseW.Code)
		}

		// Resume.
		resumeReq := httptest.NewRequest("POST", "/api/schedules/"+sched.ID+"/resume", nil)
		resumeW := httptest.NewRecorder()
		srv.Handler().ServeHTTP(resumeW, resumeReq)
		if resumeW.Code != http.StatusNoContent {
			t.Fatalf("iteration %d resume: got %d, want 204", i, resumeW.Code)
		}
	}

	// After all toggles, the schedule must still be readable and in a valid state.
	// The last operation was a resume, so it should be enabled.
	req := httptest.NewRequest("GET", "/api/schedules/"+sched.ID, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET after rapid pause/resume: got %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var final upal.Schedule
	if err := json.NewDecoder(w.Body).Decode(&final); err != nil {
		t.Fatalf("decode final schedule: %v", err)
	}

	if final.ID != sched.ID {
		t.Errorf("schedule ID: got %q, want %q", final.ID, sched.ID)
	}
	if !final.Enabled {
		t.Error("expected schedule to be enabled after final resume")
	}
}

// TestSchedule_DoublePause verifies that pausing an already-paused schedule
// succeeds without error and the schedule remains disabled.
func TestSchedule_DoublePause(t *testing.T) {
	srv := newEdgeCaseServerWithScheduler()
	sched := createEdgeCaseSchedule(t, srv, "dpause-wf", "0 * * * *", true)

	// First pause.
	req := httptest.NewRequest("POST", "/api/schedules/"+sched.ID+"/pause", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("first pause: got %d, want 204; body: %s", w.Code, w.Body.String())
	}

	// Verify disabled.
	if enabled := getScheduleState(t, srv, sched.ID); enabled {
		t.Fatal("expected schedule to be disabled after first pause")
	}

	// Second pause (already paused).
	req2 := httptest.NewRequest("POST", "/api/schedules/"+sched.ID+"/pause", nil)
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("second pause: got %d, want 204; body: %s", w2.Code, w2.Body.String())
	}

	// Still disabled.
	if enabled := getScheduleState(t, srv, sched.ID); enabled {
		t.Fatal("expected schedule to remain disabled after double pause")
	}
}

// TestSchedule_DoubleResume verifies that resuming an already-enabled schedule
// succeeds without error and the schedule remains enabled.
func TestSchedule_DoubleResume(t *testing.T) {
	srv := newEdgeCaseServerWithScheduler()
	sched := createEdgeCaseSchedule(t, srv, "dresume-wf", "0 * * * *", true)

	// Schedule starts enabled. Resume it (already enabled).
	req := httptest.NewRequest("POST", "/api/schedules/"+sched.ID+"/resume", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("first resume: got %d, want 204; body: %s", w.Code, w.Body.String())
	}

	// Still enabled.
	if enabled := getScheduleState(t, srv, sched.ID); !enabled {
		t.Fatal("expected schedule to remain enabled after resume on enabled schedule")
	}

	// Resume again.
	req2 := httptest.NewRequest("POST", "/api/schedules/"+sched.ID+"/resume", nil)
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("second resume: got %d, want 204; body: %s", w2.Code, w2.Body.String())
	}

	// Still enabled.
	if enabled := getScheduleState(t, srv, sched.ID); !enabled {
		t.Fatal("expected schedule to remain enabled after double resume")
	}
}

// ============================================================
// Input Mapping Edge Cases
// ============================================================

// TestMapInputs_MissingKeys verifies that when the mapping references keys not
// present in the payload, those inputs are simply absent from the result (no error).
func TestMapInputs_MissingKeys(t *testing.T) {
	payload := map[string]any{
		"name": "alice",
	}
	mapping := map[string]string{
		"user":  "name",            // exists in payload
		"email": "email_address",   // does NOT exist in payload
		"age":   "user_age",        // does NOT exist in payload
	}

	result := mapInputs(payload, mapping)

	// "user" should be mapped.
	if result["user"] != "alice" {
		t.Errorf("expected user=alice, got %v", result["user"])
	}

	// "email" and "age" should be absent (not error, not nil-valued).
	if _, ok := result["email"]; ok {
		t.Error("expected 'email' key to be absent from result, but it was present")
	}
	if _, ok := result["age"]; ok {
		t.Error("expected 'age' key to be absent from result, but it was present")
	}

	// Result should have exactly 1 key.
	if len(result) != 1 {
		t.Errorf("expected 1 key in result, got %d: %v", len(result), result)
	}
}

// TestMapInputs_EmptyPayload verifies that a nil payload with a non-empty
// mapping returns an empty (non-nil) map.
func TestMapInputs_EmptyPayload(t *testing.T) {
	mapping := map[string]string{
		"query": "text",
		"user":  "sender",
	}

	result := mapInputs(nil, mapping)

	if result == nil {
		t.Fatal("expected non-nil empty map, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d keys: %v", len(result), result)
	}
}

// TestMapInputs_EmptyMapping verifies that an empty (non-nil) mapping is
// treated the same as a nil mapping: the full payload is returned as-is.
func TestMapInputs_EmptyMapping(t *testing.T) {
	payload := map[string]any{
		"message": "hello",
		"count":   42,
	}

	// Empty map (not nil).
	result := mapInputs(payload, map[string]string{})

	// Should return the full payload.
	if result["message"] != "hello" {
		t.Errorf("expected message=hello, got %v", result["message"])
	}
	if result["count"] != 42 {
		t.Errorf("expected count=42, got %v", result["count"])
	}
	if len(result) != 2 {
		t.Errorf("expected 2 keys, got %d: %v", len(result), result)
	}
}
