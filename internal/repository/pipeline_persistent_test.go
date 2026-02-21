// internal/repository/pipeline_persistent_test.go
package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

var errFake = errors.New("fake db error")

// stubPipelineDB is a fake DB that records calls and returns canned data.
type stubPipelineDB struct {
	pipelines []*upal.Pipeline
	runs      []*upal.PipelineRun
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
}

func (s *stubPipelineDB) CreatePipeline(_ context.Context, p *upal.Pipeline) error {
	s.pipelines = append(s.pipelines, p)
	return s.createErr
}
func (s *stubPipelineDB) GetPipeline(_ context.Context, id string) (*upal.Pipeline, error) {
	for _, p := range s.pipelines {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, s.getErr
}
func (s *stubPipelineDB) ListPipelines(_ context.Context) ([]*upal.Pipeline, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.pipelines, nil
}
func (s *stubPipelineDB) UpdatePipeline(_ context.Context, p *upal.Pipeline) error {
	return s.updateErr
}
func (s *stubPipelineDB) DeletePipeline(_ context.Context, _ string) error {
	return s.deleteErr
}
func (s *stubPipelineDB) CreatePipelineRun(_ context.Context, r *upal.PipelineRun) error {
	s.runs = append(s.runs, r)
	return s.createErr
}
func (s *stubPipelineDB) GetPipelineRun(_ context.Context, id string) (*upal.PipelineRun, error) {
	for _, r := range s.runs {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, s.getErr
}
func (s *stubPipelineDB) ListPipelineRunsByPipeline(_ context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	var out []*upal.PipelineRun
	for _, r := range s.runs {
		if r.PipelineID == pipelineID {
			out = append(out, r)
		}
	}
	return out, nil
}
func (s *stubPipelineDB) UpdatePipelineRun(_ context.Context, _ *upal.PipelineRun) error {
	return s.updateErr
}

func newTestPipeline(id string) *upal.Pipeline {
	return &upal.Pipeline{
		ID:        id,
		Name:      "Pipeline " + id,
		Stages:    []upal.Stage{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestPersistentPipelineRepository_CreateAndGet(t *testing.T) {
	mem := repository.NewMemoryPipelineRepository()
	stub := &stubPipelineDB{}
	repo := repository.NewPersistentPipelineRepository(mem, stub)

	p := newTestPipeline("pipe-1")
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Should be in memory
	got, err := repo.Get(context.Background(), "pipe-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != "pipe-1" {
		t.Errorf("expected pipe-1, got %s", got.ID)
	}

	// DB should have been called
	if len(stub.pipelines) != 1 {
		t.Errorf("expected 1 pipeline in DB stub, got %d", len(stub.pipelines))
	}
}

func TestPersistentPipelineRepository_GetFallsBackToDb(t *testing.T) {
	mem := repository.NewMemoryPipelineRepository()
	p := newTestPipeline("pipe-db")
	stub := &stubPipelineDB{pipelines: []*upal.Pipeline{p}}
	repo := repository.NewPersistentPipelineRepository(mem, stub)

	// Memory is empty, DB has the pipeline
	got, err := repo.Get(context.Background(), "pipe-db")
	if err != nil {
		t.Fatalf("Get fallback failed: %v", err)
	}
	if got.ID != "pipe-db" {
		t.Errorf("expected pipe-db, got %s", got.ID)
	}
}

func TestPersistentPipelineRepository_ListPrefersDb(t *testing.T) {
	mem := repository.NewMemoryPipelineRepository()
	p1 := newTestPipeline("pipe-1")
	p2 := newTestPipeline("pipe-2")
	stub := &stubPipelineDB{pipelines: []*upal.Pipeline{p1, p2}}
	repo := repository.NewPersistentPipelineRepository(mem, stub)

	list, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 pipelines from DB, got %d", len(list))
	}
}

func TestPersistentPipelineRepository_ListFallsBackToMemory(t *testing.T) {
	mem := repository.NewMemoryPipelineRepository()
	p := newTestPipeline("pipe-mem")
	_ = mem.Create(context.Background(), p)
	stub := &stubPipelineDB{listErr: errFake}
	repo := repository.NewPersistentPipelineRepository(mem, stub)

	list, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List memory fallback failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != "pipe-mem" {
		t.Errorf("expected memory fallback with pipe-mem, got %v", list)
	}
}

func TestPersistentPipelineRunRepository_CreateAndListByPipeline(t *testing.T) {
	mem := repository.NewMemoryPipelineRunRepository()
	stub := &stubPipelineDB{}
	repo := repository.NewPersistentPipelineRunRepository(mem, stub)

	run := &upal.PipelineRun{
		ID:           "prun-1",
		PipelineID:   "pipe-1",
		Status:       "running",
		StageResults: map[string]*upal.StageResult{},
		StartedAt:    time.Now(),
	}
	if err := repo.Create(context.Background(), run); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	list, err := repo.ListByPipeline(context.Background(), "pipe-1")
	if err != nil {
		t.Fatalf("ListByPipeline failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 run, got %d", len(list))
	}
}
