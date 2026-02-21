// internal/api/pipelines_test.go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// noopStageExecutor completes any stage immediately.
type noopStageExecutor struct{ t string }

func (n *noopStageExecutor) Type() string { return n.t }
func (n *noopStageExecutor) Execute(_ context.Context, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	return &upal.StageResult{StageID: stage.ID, Status: "completed"}, nil
}

func newTestPipelineServer(t *testing.T) (*Server, *repository.MemoryPipelineRepository, *repository.MemoryPipelineRunRepository) {
	t.Helper()
	pipelineRepo := repository.NewMemoryPipelineRepository()
	runRepo := repository.NewMemoryPipelineRunRepository()
	pipelineSvc := services.NewPipelineService(pipelineRepo, runRepo)
	runner := services.NewPipelineRunner(runRepo)
	runner.RegisterExecutor(&noopStageExecutor{"workflow"})

	srv := &Server{}
	srv.SetPipelineService(pipelineSvc)
	srv.SetPipelineRunner(runner)
	return srv, pipelineRepo, runRepo
}

func TestApprovePipelineRun(t *testing.T) {
	srv, pipelineRepo, runRepo := newTestPipelineServer(t)

	// Seed a pipeline
	pipeline := &upal.Pipeline{
		ID:   "pipe-1",
		Name: "Test",
		Stages: []upal.Stage{
			{ID: "s1", Type: "workflow"},
			{ID: "s2", Type: "approval"},
			{ID: "s3", Type: "workflow"},
		},
	}
	pipelineRepo.Create(context.Background(), pipeline)

	// Seed a waiting run
	run := &upal.PipelineRun{
		ID:           "prun-1",
		PipelineID:   "pipe-1",
		Status:       "waiting",
		CurrentStage: "s2",
		StageResults: map[string]*upal.StageResult{
			"s1": {StageID: "s1", Status: "completed", StartedAt: time.Now()},
			"s2": {StageID: "s2", Status: "waiting", StartedAt: time.Now()},
		},
		StartedAt: time.Now(),
	}
	runRepo.Create(context.Background(), run)

	req := httptest.NewRequest(http.MethodPost, "/api/pipelines/pipe-1/runs/prun-1/approve", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "pipe-1")
	rctx.URLParams.Add("runId", "prun-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.approvePipelineRun(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp upal.PipelineRun
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "running" {
		t.Errorf("expected run status 'running' in response, got %q", resp.Status)
	}
}

func TestRejectPipelineRun(t *testing.T) {
	srv, pipelineRepo, runRepo := newTestPipelineServer(t)

	pipeline := &upal.Pipeline{
		ID:   "pipe-2",
		Name: "Reject Test",
		Stages: []upal.Stage{
			{ID: "s1", Type: "approval"},
		},
	}
	pipelineRepo.Create(context.Background(), pipeline)

	run := &upal.PipelineRun{
		ID:           "prun-2",
		PipelineID:   "pipe-2",
		Status:       "waiting",
		CurrentStage: "s1",
		StageResults: map[string]*upal.StageResult{
			"s1": {StageID: "s1", Status: "waiting", StartedAt: time.Now()},
		},
		StartedAt: time.Now(),
	}
	runRepo.Create(context.Background(), run)

	req := httptest.NewRequest(http.MethodPost, "/api/pipelines/pipe-2/runs/prun-2/reject", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "pipe-2")
	rctx.URLParams.Add("runId", "prun-2")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.rejectPipelineRun(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify persisted as failed
	updated, _ := runRepo.Get(context.Background(), "prun-2")
	if updated.Status != "failed" {
		t.Errorf("expected persisted status 'failed', got %q", updated.Status)
	}
}

func TestRejectPipelineRun_NotWaiting(t *testing.T) {
	srv, pipelineRepo, runRepo := newTestPipelineServer(t)

	pipeline := &upal.Pipeline{ID: "pipe-4", Name: "P4", Stages: []upal.Stage{{ID: "s1", Type: "workflow"}}}
	pipelineRepo.Create(context.Background(), pipeline)

	run := &upal.PipelineRun{ID: "prun-4", PipelineID: "pipe-4", Status: "completed", StageResults: map[string]*upal.StageResult{}}
	runRepo.Create(context.Background(), run)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "pipe-4")
	rctx.URLParams.Add("runId", "prun-4")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.rejectPipelineRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestApprovePipelineRun_NotWaiting(t *testing.T) {
	srv, pipelineRepo, runRepo := newTestPipelineServer(t)

	pipeline := &upal.Pipeline{ID: "pipe-3", Name: "P3", Stages: []upal.Stage{{ID: "s1", Type: "workflow"}}}
	pipelineRepo.Create(context.Background(), pipeline)

	run := &upal.PipelineRun{ID: "prun-3", PipelineID: "pipe-3", Status: "completed", StageResults: map[string]*upal.StageResult{}}
	runRepo.Create(context.Background(), run)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "pipe-3")
	rctx.URLParams.Add("runId", "prun-3")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.approvePipelineRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
