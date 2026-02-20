package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/soochol/upal/internal/repository"
)

// newTestServerWithTriggers creates a test server with a MemoryTriggerRepository configured.
func newTestServerWithTriggers() *Server {
	srv := newTestServer()
	srv.SetTriggerRepository(repository.NewMemoryTriggerRepository())
	return srv
}

// createTriggerHelper sends a POST /api/triggers request and returns the recorder.
func createTriggerHelper(t *testing.T, srv *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/triggers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	return w
}

func TestCreateTrigger(t *testing.T) {
	srv := newTestServerWithTriggers()

	body := `{"workflow_name": "my-workflow"}`
	w := createTriggerHelper(t, srv, body)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify webhook_url is present.
	if _, ok := resp["webhook_url"]; !ok {
		t.Fatal("response missing webhook_url")
	}

	// Decode the trigger object.
	var trigger struct {
		ID           string `json:"id"`
		WorkflowName string `json:"workflow_name"`
		Type         string `json:"type"`
		Enabled      bool   `json:"enabled"`
		Config       struct {
			Secret string `json:"secret"`
		} `json:"config"`
	}
	if err := json.Unmarshal(resp["trigger"], &trigger); err != nil {
		t.Fatalf("failed to decode trigger: %v", err)
	}

	// ID starts with "trig-".
	if !strings.HasPrefix(trigger.ID, "trig-") {
		t.Errorf("expected ID prefix 'trig-', got %q", trigger.ID)
	}

	// Type is webhook.
	if trigger.Type != "webhook" {
		t.Errorf("expected type 'webhook', got %q", trigger.Type)
	}

	// Enabled is true.
	if !trigger.Enabled {
		t.Error("expected enabled=true")
	}

	// Secret auto-generated with "whsec_" prefix.
	if !strings.HasPrefix(trigger.Config.Secret, "whsec_") {
		t.Errorf("expected secret prefix 'whsec_', got %q", trigger.Config.Secret)
	}

	// webhook_url contains the trigger ID.
	var webhookURL string
	if err := json.Unmarshal(resp["webhook_url"], &webhookURL); err != nil {
		t.Fatalf("failed to decode webhook_url: %v", err)
	}
	expectedURL := "/api/hooks/" + trigger.ID
	if webhookURL != expectedURL {
		t.Errorf("expected webhook_url %q, got %q", expectedURL, webhookURL)
	}
}

func TestCreateTrigger_MissingWorkflow(t *testing.T) {
	srv := newTestServerWithTriggers()

	// Empty workflow_name.
	body := `{"workflow_name": ""}`
	w := createTriggerHelper(t, srv, body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	// Also test with no workflow_name field at all.
	body2 := `{}`
	w2 := createTriggerHelper(t, srv, body2)

	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing field, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestCreateTrigger_CustomSecret(t *testing.T) {
	srv := newTestServerWithTriggers()

	body := `{"workflow_name": "my-wf", "config": {"secret": "my-custom-secret"}}`
	w := createTriggerHelper(t, srv, body)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	var trigger struct {
		Config struct {
			Secret string `json:"secret"`
		} `json:"config"`
	}
	if err := json.Unmarshal(resp["trigger"], &trigger); err != nil {
		t.Fatalf("failed to decode trigger: %v", err)
	}

	// Custom secret should be preserved (not overridden).
	if trigger.Config.Secret != "my-custom-secret" {
		t.Errorf("expected custom secret 'my-custom-secret', got %q", trigger.Config.Secret)
	}
}

func TestCreateTrigger_NoTriggerRepo(t *testing.T) {
	// Use base test server without trigger repo.
	srv := newTestServer()

	body := `{"workflow_name": "my-wf"}`
	w := createTriggerHelper(t, srv, body)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateTrigger_WithInputMapping(t *testing.T) {
	srv := newTestServerWithTriggers()

	body := `{
		"workflow_name": "my-wf",
		"config": {
			"input_mapping": {
				"$.payload.name": "user_name",
				"$.payload.email": "user_email"
			}
		}
	}`
	w := createTriggerHelper(t, srv, body)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	var trigger struct {
		Config struct {
			Secret       string            `json:"secret"`
			InputMapping map[string]string `json:"input_mapping"`
		} `json:"config"`
	}
	if err := json.Unmarshal(resp["trigger"], &trigger); err != nil {
		t.Fatalf("failed to decode trigger: %v", err)
	}

	if len(trigger.Config.InputMapping) != 2 {
		t.Fatalf("expected 2 input mappings, got %d", len(trigger.Config.InputMapping))
	}
	if trigger.Config.InputMapping["$.payload.name"] != "user_name" {
		t.Errorf("expected mapping '$.payload.name' -> 'user_name', got %q", trigger.Config.InputMapping["$.payload.name"])
	}
	if trigger.Config.InputMapping["$.payload.email"] != "user_email" {
		t.Errorf("expected mapping '$.payload.email' -> 'user_email', got %q", trigger.Config.InputMapping["$.payload.email"])
	}

	// Secret should still be auto-generated since none was provided.
	if !strings.HasPrefix(trigger.Config.Secret, "whsec_") {
		t.Errorf("expected auto-generated secret with 'whsec_' prefix, got %q", trigger.Config.Secret)
	}
}

func TestListTriggers(t *testing.T) {
	srv := newTestServerWithTriggers()

	// Create 2 triggers for "wf-a".
	createTriggerHelper(t, srv, `{"workflow_name": "wf-a"}`)
	createTriggerHelper(t, srv, `{"workflow_name": "wf-a"}`)
	// Create 1 trigger for "wf-b".
	createTriggerHelper(t, srv, `{"workflow_name": "wf-b"}`)

	// List triggers for "wf-a".
	req := httptest.NewRequest("GET", "/api/workflows/wf-a/triggers", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var triggers []json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &triggers); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(triggers) != 2 {
		t.Fatalf("expected 2 triggers for wf-a, got %d", len(triggers))
	}

	// List triggers for "wf-b".
	req2 := httptest.NewRequest("GET", "/api/workflows/wf-b/triggers", nil)
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var triggersB []json.RawMessage
	if err := json.Unmarshal(w2.Body.Bytes(), &triggersB); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(triggersB) != 1 {
		t.Fatalf("expected 1 trigger for wf-b, got %d", len(triggersB))
	}
}

func TestListTriggers_Empty(t *testing.T) {
	srv := newTestServerWithTriggers()

	req := httptest.NewRequest("GET", "/api/workflows/nonexistent/triggers", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should be a valid JSON array (possibly null for nil slice, but let's check it decodes).
	var triggers []json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &triggers); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(triggers) != 0 {
		t.Fatalf("expected 0 triggers, got %d", len(triggers))
	}
}

func TestListTriggers_NoTriggerRepo(t *testing.T) {
	// Without trigger repo, listTriggers returns empty array.
	srv := newTestServer()

	req := httptest.NewRequest("GET", "/api/workflows/any-wf/triggers", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var triggers []json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &triggers); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(triggers) != 0 {
		t.Fatalf("expected empty array, got %d items", len(triggers))
	}
}

func TestDeleteTrigger(t *testing.T) {
	srv := newTestServerWithTriggers()

	// Create a trigger first.
	w := createTriggerHelper(t, srv, `{"workflow_name": "del-wf"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d", w.Code)
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	var trigger struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp["trigger"], &trigger); err != nil {
		t.Fatalf("failed to decode trigger: %v", err)
	}

	// Delete it.
	delReq := httptest.NewRequest("DELETE", "/api/triggers/"+trigger.ID, nil)
	delW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", delW.Code, delW.Body.String())
	}

	// Verify it's gone by listing triggers for the workflow.
	listReq := httptest.NewRequest("GET", "/api/workflows/del-wf/triggers", nil)
	listW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW, listReq)

	var triggers []json.RawMessage
	if err := json.Unmarshal(listW.Body.Bytes(), &triggers); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}

	if len(triggers) != 0 {
		t.Fatalf("expected 0 triggers after delete, got %d", len(triggers))
	}
}

func TestDeleteTrigger_NotFound(t *testing.T) {
	srv := newTestServerWithTriggers()

	req := httptest.NewRequest("DELETE", "/api/triggers/trig-nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTrigger_NoTriggerRepo(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("DELETE", "/api/triggers/trig-anything", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}
