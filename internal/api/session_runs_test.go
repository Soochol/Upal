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
