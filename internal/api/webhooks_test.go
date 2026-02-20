package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/session"
)

// newTestServerWithWebhook creates a Server wired with an in-memory trigger
// repository and workflow service, suitable for webhook handler integration tests.
func newTestServerWithWebhook() (*Server, repository.TriggerRepository) {
	sessionSvc := session.InMemoryService()
	wfRepo := repository.NewMemory()
	wfSvc := services.NewWorkflowService(wfRepo, nil, sessionSvc, nil)
	trigRepo := repository.NewMemoryTriggerRepository()
	srv := NewServer(nil, wfSvc, wfRepo, nil)
	srv.SetTriggerRepository(trigRepo)
	return srv, trigRepo
}

// seedWorkflow creates a minimal workflow via the API so that the webhook
// handler's Lookup call succeeds.
func seedWorkflow(t *testing.T, srv *Server, name string) {
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
		t.Fatalf("seedWorkflow: got status %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

// signPayload computes the HMAC-SHA256 hex signature for a given payload and secret.
func signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyHMAC(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"message":"hello"}`)

	// Generate valid signature.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		payload   []byte
		secret    string
		signature string
		valid     bool
	}{
		{"valid signature", payload, secret, validSig, true},
		{"wrong signature", payload, secret, "deadbeef", false},
		{"empty signature", payload, secret, "", false},
		{"wrong secret", payload, "other-secret", validSig, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := verifyHMAC(tt.payload, tt.secret, tt.signature)
			if got != tt.valid {
				t.Errorf("verifyHMAC() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestMapInputs(t *testing.T) {
	payload := map[string]any{
		"message": "hello world",
		"user":    "alice",
		"count":   42,
	}

	t.Run("with mapping", func(t *testing.T) {
		mapping := map[string]string{
			"query":    "message",
			"username": "user",
		}
		inputs := mapInputs(payload, mapping)
		if inputs["query"] != "hello world" {
			t.Errorf("expected query=hello world, got %v", inputs["query"])
		}
		if inputs["username"] != "alice" {
			t.Errorf("expected username=alice, got %v", inputs["username"])
		}
		if _, ok := inputs["count"]; ok {
			t.Error("count should not be in inputs (not in mapping)")
		}
	})

	t.Run("without mapping", func(t *testing.T) {
		inputs := mapInputs(payload, nil)
		if inputs["message"] != "hello world" {
			t.Errorf("expected full payload passthrough, got %v", inputs)
		}
	})
}

func TestHandleWebhook_Success(t *testing.T) {
	srv, trigRepo := newTestServerWithWebhook()
	seedWorkflow(t, srv, "test-wf")

	secret := "my-secret"
	trigger := &upal.Trigger{
		ID:           "trig_test123",
		WorkflowName: "test-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{Secret: secret},
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	payload := []byte(`{"message":"hello"}`)
	sig := signPayload(payload, secret)

	req := httptest.NewRequest("POST", "/api/hooks/trig_test123", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", sig)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["status"] != "accepted" {
		t.Errorf("status field: got %q, want %q", resp["status"], "accepted")
	}
	if resp["trigger"] != "trig_test123" {
		t.Errorf("trigger field: got %q, want %q", resp["trigger"], "trig_test123")
	}
}

func TestHandleWebhook_TriggerNotFound(t *testing.T) {
	srv, _ := newTestServerWithWebhook()

	req := httptest.NewRequest("POST", "/api/hooks/nonexistent", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404; body: %s", w.Code, w.Body.String())
	}
}

func TestHandleWebhook_TriggerDisabled(t *testing.T) {
	srv, trigRepo := newTestServerWithWebhook()

	trigger := &upal.Trigger{
		ID:           "trig_disabled",
		WorkflowName: "test-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{},
		Enabled:      false,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/hooks/trig_disabled", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403; body: %s", w.Code, w.Body.String())
	}
}

func TestHandleWebhook_InvalidSignature(t *testing.T) {
	srv, trigRepo := newTestServerWithWebhook()

	trigger := &upal.Trigger{
		ID:           "trig_secret",
		WorkflowName: "test-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{Secret: "correct-secret"},
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	payload := []byte(`{"message":"hello"}`)
	wrongSig := signPayload(payload, "wrong-secret")

	req := httptest.NewRequest("POST", "/api/hooks/trig_secret", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", wrongSig)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401; body: %s", w.Code, w.Body.String())
	}
}

func TestHandleWebhook_NoSecret(t *testing.T) {
	srv, trigRepo := newTestServerWithWebhook()
	seedWorkflow(t, srv, "test-wf")

	trigger := &upal.Trigger{
		ID:           "trig_nosecret",
		WorkflowName: "test-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{Secret: ""},
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	payload := []byte(`{"data":"test"}`)
	// No X-Webhook-Signature header — should still succeed because secret is empty.
	req := httptest.NewRequest("POST", "/api/hooks/trig_nosecret", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["status"] != "accepted" {
		t.Errorf("status field: got %q, want %q", resp["status"], "accepted")
	}
}

func TestHandleWebhook_EmptyBody(t *testing.T) {
	srv, trigRepo := newTestServerWithWebhook()
	seedWorkflow(t, srv, "test-wf")

	trigger := &upal.Trigger{
		ID:           "trig_emptybody",
		WorkflowName: "test-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{},
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	// Send request with empty body.
	req := httptest.NewRequest("POST", "/api/hooks/trig_emptybody", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202; body: %s", w.Code, w.Body.String())
	}
}

func TestHandleWebhook_NoTriggerRepo(t *testing.T) {
	// Create a server without setting the trigger repository.
	sessionSvc := session.InMemoryService()
	wfRepo := repository.NewMemory()
	wfSvc := services.NewWorkflowService(wfRepo, nil, sessionSvc, nil)
	srv := NewServer(nil, wfSvc, wfRepo, nil)
	// Deliberately do NOT call srv.SetTriggerRepository().

	req := httptest.NewRequest("POST", "/api/hooks/any-id", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want 503; body: %s", w.Code, w.Body.String())
	}
}

func TestHandleWebhook_InputMapping(t *testing.T) {
	srv, trigRepo := newTestServerWithWebhook()
	seedWorkflow(t, srv, "mapped-wf")

	trigger := &upal.Trigger{
		ID:           "trig_mapped",
		WorkflowName: "mapped-wf",
		Type:         upal.TriggerWebhook,
		Config: upal.TriggerConfig{
			Secret: "map-secret",
			InputMapping: map[string]string{
				"query": "text",
				"user":  "sender",
			},
		},
		Enabled:   true,
		CreatedAt: time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	payload := []byte(`{"text":"hello world","sender":"alice","extra":"ignored"}`)
	sig := signPayload(payload, "map-secret")

	req := httptest.NewRequest("POST", "/api/hooks/trig_mapped", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", sig)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["status"] != "accepted" {
		t.Errorf("status field: got %q, want %q", resp["status"], "accepted")
	}
	if resp["trigger"] != "trig_mapped" {
		t.Errorf("trigger field: got %q, want %q", resp["trigger"], "trig_mapped")
	}
}
