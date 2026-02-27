package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// newTestRunServer creates a Server wired for session-run tests.
func newTestRunServer() (*Server, *services.SessionService) {
	srv := newTestServer()

	sessRepo := repository.NewMemorySessionRepository()
	runRepo := repository.NewMemorySessionRunRepository()
	fetchRepo := repository.NewMemorySourceFetchRepository()
	analysisRepo := repository.NewMemoryLLMAnalysisRepository()
	publishedRepo := repository.NewMemoryPublishedContentRepository()
	surgeRepo := repository.NewMemorySurgeEventRepository()
	wfRunRepo := repository.NewMemoryWorkflowRunRepository()

	sessSvc := services.NewSessionService(sessRepo)
	runSvc := services.NewRunService(runRepo, sessRepo, fetchRepo, analysisRepo, publishedRepo, surgeRepo, wfRunRepo)
	srv.SetSessionService(sessSvc)
	srv.SetRunService(runSvc)

	return srv, sessSvc
}

func TestDeleteNewRun(t *testing.T) {
	srv, sessSvc := newTestRunServer()
	h := srv.Handler()

	// Create a session.
	sess, err := sessSvc.Create(t.Context(), &upal.Session{Name: "test-session"})
	if err != nil {
		t.Fatal(err)
	}

	// Create a run via API.
	createReq := httptest.NewRequest("POST", "/api/sessions/"+sess.ID+"/runs", bytes.NewReader([]byte("{}")))
	createReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, createReq)
	if w.Code != http.StatusCreated {
		t.Fatalf("create run: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var run upal.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}

	// Verify the run exists.
	getReq := httptest.NewRequest("GET", "/api/session-runs/"+run.ID, nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, getReq)
	if w.Code != http.StatusOK {
		t.Fatalf("get run: expected 200, got %d", w.Code)
	}

	// Delete the run.
	delReq := httptest.NewRequest("DELETE", "/api/session-runs/"+run.ID, nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, delReq)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete run: expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone.
	getReq2 := httptest.NewRequest("GET", "/api/session-runs/"+run.ID, nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, getReq2)
	if w.Code != http.StatusNotFound {
		t.Fatalf("after delete: expected 404, got %d", w.Code)
	}
}

func TestToggleRunSchedule_Once(t *testing.T) {
	srv, sessSvc := newTestRunServer()
	h := srv.Handler()

	sess, err := sessSvc.Create(t.Context(), &upal.Session{Name: "test-session"})
	if err != nil {
		t.Fatal(err)
	}

	// Create a run with @once schedule and some content.
	body := `{"name":"once-run","schedule":"@once","context":{"prompt":"test task"}}`
	createReq := httptest.NewRequest("POST", "/api/sessions/"+sess.ID+"/runs", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, createReq)
	if w.Code != http.StatusCreated {
		t.Fatalf("create run: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var run upal.Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.Schedule != "@once" {
		t.Fatalf("expected schedule @once, got %q", run.Schedule)
	}
	if run.Status != upal.SessionRunDraft {
		t.Fatalf("expected draft, got %s", run.Status)
	}

	// Toggle active=true on @once run — should start collection, keep schedule_active false.
	toggleBody := `{"active":true}`
	toggleReq := httptest.NewRequest("POST", "/api/session-runs/"+run.ID+"/schedule/toggle", bytes.NewReader([]byte(toggleBody)))
	toggleReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, toggleReq)
	if w.Code != http.StatusOK {
		t.Fatalf("toggle: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var toggled upal.Run
	if err := json.NewDecoder(w.Body).Decode(&toggled); err != nil {
		t.Fatal(err)
	}
	// Status should be collecting (triggered immediately).
	if toggled.Status != upal.SessionRunCollecting {
		t.Fatalf("expected collecting, got %s", toggled.Status)
	}
	// schedule_active stays false (one-shot, no repeat).
	if toggled.ScheduleActive {
		t.Fatal("expected schedule_active to remain false for @once")
	}
}

func TestDeleteNewRun_NotFound(t *testing.T) {
	srv, _ := newTestRunServer()
	h := srv.Handler()

	delReq := httptest.NewRequest("DELETE", "/api/session-runs/nonexistent", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, delReq)
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Fatalf("delete nonexistent: expected 404 or 500, got %d", w.Code)
	}
}
