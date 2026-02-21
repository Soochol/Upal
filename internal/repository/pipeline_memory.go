// internal/repository/pipeline_memory.go
package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/soochol/upal/internal/upal"
)

// MemoryPipelineRepository implements PipelineRepository in-memory.
type MemoryPipelineRepository struct {
	mu        sync.RWMutex
	pipelines map[string]*upal.Pipeline
}

func NewMemoryPipelineRepository() *MemoryPipelineRepository {
	return &MemoryPipelineRepository{pipelines: make(map[string]*upal.Pipeline)}
}

func (r *MemoryPipelineRepository) Create(_ context.Context, p *upal.Pipeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.pipelines[p.ID]; exists {
		return fmt.Errorf("pipeline %q already exists", p.ID)
	}
	r.pipelines[p.ID] = p
	return nil
}

func (r *MemoryPipelineRepository) Get(_ context.Context, id string) (*upal.Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.pipelines[id]
	if !ok {
		return nil, fmt.Errorf("pipeline %q not found", id)
	}
	return p, nil
}

func (r *MemoryPipelineRepository) List(_ context.Context) ([]*upal.Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*upal.Pipeline, 0, len(r.pipelines))
	for _, p := range r.pipelines {
		out = append(out, p)
	}
	return out, nil
}

func (r *MemoryPipelineRepository) Update(_ context.Context, p *upal.Pipeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.pipelines[p.ID]; !exists {
		return fmt.Errorf("pipeline %q not found", p.ID)
	}
	r.pipelines[p.ID] = p
	return nil
}

func (r *MemoryPipelineRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.pipelines[id]; !exists {
		return fmt.Errorf("pipeline %q not found", id)
	}
	delete(r.pipelines, id)
	return nil
}

// MemoryPipelineRunRepository implements PipelineRunRepository in-memory.
type MemoryPipelineRunRepository struct {
	mu   sync.RWMutex
	runs map[string]*upal.PipelineRun
}

func NewMemoryPipelineRunRepository() *MemoryPipelineRunRepository {
	return &MemoryPipelineRunRepository{runs: make(map[string]*upal.PipelineRun)}
}

func (r *MemoryPipelineRunRepository) Create(_ context.Context, run *upal.PipelineRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[run.ID] = run
	return nil
}

func (r *MemoryPipelineRunRepository) Get(_ context.Context, id string) (*upal.PipelineRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.runs[id]
	if !ok {
		return nil, fmt.Errorf("pipeline run %q not found", id)
	}
	return run, nil
}

func (r *MemoryPipelineRunRepository) ListByPipeline(_ context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*upal.PipelineRun
	for _, run := range r.runs {
		if run.PipelineID == pipelineID {
			out = append(out, run)
		}
	}
	return out, nil
}

func (r *MemoryPipelineRunRepository) Update(_ context.Context, run *upal.PipelineRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.runs[run.ID]; !exists {
		return fmt.Errorf("pipeline run %q not found", run.ID)
	}
	r.runs[run.ID] = run
	return nil
}
