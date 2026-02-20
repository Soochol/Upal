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
	"google.golang.org/adk/session"
)

// newTestServerWithScheduler creates a Server wired up with a full
// SchedulerService and all its dependencies backed by in-memory stores.
func newTestServerWithScheduler() *Server {
	sessionSvc := session.InMemoryService()
	repo := repository.NewMemory()
	wfSvc := services.NewWorkflowService(repo, nil, sessionSvc, nil)
	srv := NewServer(nil, wfSvc, repo, nil)

	schedRepo := repository.NewMemoryScheduleRepository()
	triggerRepo := repository.NewMemoryTriggerRepository()
	runRepo := repository.NewMemoryRunRepository()
	runHistorySvc := services.NewRunHistoryService(runRepo)
	limiter := services.NewConcurrencyLimiter(upal.ConcurrencyLimits{GlobalMax: 10, PerWorkflow: 3})
	retryExec := services.NewRetryExecutor(wfSvc, runHistorySvc)
	schedulerSvc := services.NewSchedulerService(schedRepo, wfSvc, retryExec, limiter, runHistorySvc)

	srv.SetSchedulerService(schedulerSvc)
	srv.SetTriggerRepository(triggerRepo)
	srv.SetConcurrencyLimiter(limiter)
	srv.SetRetryExecutor(retryExec)
	srv.SetRunHistoryService(runHistorySvc)

	return srv
}

// createScheduleViaAPI is a helper that POSTs a schedule and returns the
// recorded response. It fails the test on JSON encoding errors.
func createScheduleViaAPI(t *testing.T, srv *Server, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal schedule body: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	return w
}

// decodeSchedule decodes a JSON response body into a upal.Schedule.
func decodeSchedule(t *testing.T, w *httptest.ResponseRecorder) upal.Schedule {
	t.Helper()
	var s upal.Schedule
	if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	return s
}

// --- Create ---

func TestCreateSchedule(t *testing.T) {
	srv := newTestServerWithScheduler()

	w := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "my-workflow",
		"cron_expr":     "*/5 * * * *",
		"enabled":       true,
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	s := decodeSchedule(t, w)

	if s.ID == "" {
		t.Error("expected non-empty ID")
	}
	if s.WorkflowName != "my-workflow" {
		t.Errorf("workflow_name: got %q, want %q", s.WorkflowName, "my-workflow")
	}
	if s.CronExpr != "*/5 * * * *" {
		t.Errorf("cron_expr: got %q, want %q", s.CronExpr, "*/5 * * * *")
	}
	if s.NextRunAt.IsZero() {
		t.Error("expected NextRunAt to be set")
	}
	if s.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if s.Timezone == "" {
		t.Error("expected Timezone to default to a value")
	}
}

func TestCreateSchedule_MissingFields(t *testing.T) {
	srv := newTestServerWithScheduler()

	// Missing workflow_name.
	w := createScheduleViaAPI(t, srv, map[string]any{
		"cron_expr": "*/5 * * * *",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing workflow_name: expected 400, got %d: %s", w.Code, w.Body.String())
	}

	// Missing cron_expr.
	w = createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "my-workflow",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing cron_expr: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSchedule_InvalidCron(t *testing.T) {
	srv := newTestServerWithScheduler()

	w := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "my-workflow",
		"cron_expr":     "not-a-cron",
		"enabled":       true,
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid cron: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSchedule_NoScheduler(t *testing.T) {
	// Use a plain server without scheduler service.
	srv := newTestServer()

	w := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "my-workflow",
		"cron_expr":     "*/5 * * * *",
	})

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no scheduler: expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// --- List ---

func TestListSchedules(t *testing.T) {
	srv := newTestServerWithScheduler()

	// Create two schedules.
	w1 := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "wf-1",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	if w1.Code != http.StatusCreated {
		t.Fatalf("create 1: got %d", w1.Code)
	}

	w2 := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "wf-2",
		"cron_expr":     "30 * * * *",
		"enabled":       true,
	})
	if w2.Code != http.StatusCreated {
		t.Fatalf("create 2: got %d", w2.Code)
	}

	// List all.
	req := httptest.NewRequest("GET", "/api/schedules", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var schedules []upal.Schedule
	if err := json.NewDecoder(w.Body).Decode(&schedules); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(schedules) != 2 {
		t.Fatalf("expected 2 schedules, got %d", len(schedules))
	}
}

func TestListSchedules_Empty(t *testing.T) {
	srv := newTestServerWithScheduler()

	req := httptest.NewRequest("GET", "/api/schedules", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list empty: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Must be a JSON array, not null.
	body := w.Body.String()
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body[0] != '[' {
		t.Fatalf("expected JSON array, got: %s", body)
	}

	var schedules []upal.Schedule
	if err := json.Unmarshal(raw, &schedules); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(schedules) != 0 {
		t.Fatalf("expected 0 schedules, got %d", len(schedules))
	}
}

func TestListSchedules_NoScheduler(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("GET", "/api/schedules", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list no scheduler: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if body[0] != '[' {
		t.Fatalf("expected JSON array, got: %s", body)
	}
}

// --- Get ---

func TestGetSchedule(t *testing.T) {
	srv := newTestServerWithScheduler()

	// Create a schedule.
	cw := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "get-wf",
		"cron_expr":     "0 12 * * *",
		"enabled":       true,
	})
	if cw.Code != http.StatusCreated {
		t.Fatalf("create: got %d", cw.Code)
	}
	created := decodeSchedule(t, cw)

	// GET by ID.
	req := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got := decodeSchedule(t, w)
	if got.ID != created.ID {
		t.Errorf("ID: got %q, want %q", got.ID, created.ID)
	}
	if got.WorkflowName != "get-wf" {
		t.Errorf("WorkflowName: got %q, want %q", got.WorkflowName, "get-wf")
	}
	if got.CronExpr != "0 12 * * *" {
		t.Errorf("CronExpr: got %q, want %q", got.CronExpr, "0 12 * * *")
	}
}

func TestGetSchedule_NotFound(t *testing.T) {
	srv := newTestServerWithScheduler()

	req := httptest.NewRequest("GET", "/api/schedules/nonexistent-id", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("not found: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetSchedule_NoScheduler(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("GET", "/api/schedules/some-id", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("no scheduler get: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Update ---

func TestUpdateSchedule(t *testing.T) {
	srv := newTestServerWithScheduler()

	// Create a schedule.
	cw := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "update-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	if cw.Code != http.StatusCreated {
		t.Fatalf("create: got %d", cw.Code)
	}
	created := decodeSchedule(t, cw)

	// Update the cron expression.
	updateBody, _ := json.Marshal(map[string]any{
		"workflow_name": "update-wf",
		"cron_expr":     "30 * * * *",
		"enabled":       true,
	})
	req := httptest.NewRequest("PUT", "/api/schedules/"+created.ID, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	updated := decodeSchedule(t, w)
	if updated.ID != created.ID {
		t.Errorf("ID changed: got %q, want %q", updated.ID, created.ID)
	}
	if updated.CronExpr != "30 * * * *" {
		t.Errorf("CronExpr: got %q, want %q", updated.CronExpr, "30 * * * *")
	}
}

func TestUpdateSchedule_NoScheduler(t *testing.T) {
	srv := newTestServer()

	body, _ := json.Marshal(map[string]any{
		"workflow_name": "wf",
		"cron_expr":     "0 * * * *",
	})
	req := httptest.NewRequest("PUT", "/api/schedules/some-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no scheduler update: expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Delete ---

func TestDeleteSchedule(t *testing.T) {
	srv := newTestServerWithScheduler()

	// Create a schedule.
	cw := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "delete-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	if cw.Code != http.StatusCreated {
		t.Fatalf("create: got %d", cw.Code)
	}
	created := decodeSchedule(t, cw)

	// Delete it.
	req := httptest.NewRequest("DELETE", "/api/schedules/"+created.ID, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone via GET.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Fatalf("after delete GET: expected 404, got %d: %s", getW.Code, getW.Body.String())
	}
}

func TestDeleteSchedule_NoScheduler(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("DELETE", "/api/schedules/some-id", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no scheduler delete: expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Pause ---

func TestPauseSchedule(t *testing.T) {
	srv := newTestServerWithScheduler()

	// Create an enabled schedule.
	cw := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "pause-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	if cw.Code != http.StatusCreated {
		t.Fatalf("create: got %d", cw.Code)
	}
	created := decodeSchedule(t, cw)

	// Pause it.
	req := httptest.NewRequest("POST", "/api/schedules/"+created.ID+"/pause", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("pause: expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPauseSchedule_NoScheduler(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("POST", "/api/schedules/some-id/pause", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no scheduler pause: expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Resume ---

func TestResumeSchedule(t *testing.T) {
	srv := newTestServerWithScheduler()

	// Create an enabled schedule, then pause it, then resume.
	cw := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "resume-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	if cw.Code != http.StatusCreated {
		t.Fatalf("create: got %d", cw.Code)
	}
	created := decodeSchedule(t, cw)

	// Pause first.
	pauseReq := httptest.NewRequest("POST", "/api/schedules/"+created.ID+"/pause", nil)
	pauseW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(pauseW, pauseReq)
	if pauseW.Code != http.StatusNoContent {
		t.Fatalf("pause: expected 204, got %d", pauseW.Code)
	}

	// Resume.
	resumeReq := httptest.NewRequest("POST", "/api/schedules/"+created.ID+"/resume", nil)
	resumeW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(resumeW, resumeReq)

	if resumeW.Code != http.StatusNoContent {
		t.Fatalf("resume: expected 204, got %d: %s", resumeW.Code, resumeW.Body.String())
	}
}

func TestResumeSchedule_NoScheduler(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("POST", "/api/schedules/some-id/resume", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no scheduler resume: expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Lifecycle ---

func TestPauseResume_Lifecycle(t *testing.T) {
	srv := newTestServerWithScheduler()

	// Step 1: Create an enabled schedule.
	cw := createScheduleViaAPI(t, srv, map[string]any{
		"workflow_name": "lifecycle-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	if cw.Code != http.StatusCreated {
		t.Fatalf("create: got %d", cw.Code)
	}
	created := decodeSchedule(t, cw)
	if !created.Enabled {
		t.Fatal("expected schedule to be enabled after creation")
	}

	// Step 2: Pause it.
	pauseReq := httptest.NewRequest("POST", "/api/schedules/"+created.ID+"/pause", nil)
	pauseW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(pauseW, pauseReq)
	if pauseW.Code != http.StatusNoContent {
		t.Fatalf("pause: expected 204, got %d", pauseW.Code)
	}

	// Step 3: Verify it is disabled via GET.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get after pause: expected 200, got %d", getW.Code)
	}
	paused := decodeSchedule(t, getW)
	if paused.Enabled {
		t.Error("expected schedule to be disabled after pause")
	}

	// Step 4: Resume it.
	resumeReq := httptest.NewRequest("POST", "/api/schedules/"+created.ID+"/resume", nil)
	resumeW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(resumeW, resumeReq)
	if resumeW.Code != http.StatusNoContent {
		t.Fatalf("resume: expected 204, got %d", resumeW.Code)
	}

	// Step 5: Verify it is enabled again via GET.
	getReq2 := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW2, getReq2)
	if getW2.Code != http.StatusOK {
		t.Fatalf("get after resume: expected 200, got %d", getW2.Code)
	}
	resumed := decodeSchedule(t, getW2)
	if !resumed.Enabled {
		t.Error("expected schedule to be enabled after resume")
	}
}
