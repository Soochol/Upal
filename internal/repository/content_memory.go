package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

// --- ContentSession ---

type MemoryContentSessionRepository struct {
	store *memstore.Store[*upal.ContentSession]
}

func NewMemoryContentSessionRepository() *MemoryContentSessionRepository {
	return &MemoryContentSessionRepository{
		store: memstore.New(func(s *upal.ContentSession) string { return s.ID }),
	}
}

func (r *MemoryContentSessionRepository) Create(ctx context.Context, s *upal.ContentSession) error {
	if r.store.Has(ctx, s.ID) {
		return fmt.Errorf("content session %q already exists", s.ID)
	}
	return r.store.Set(ctx, s)
}

func (r *MemoryContentSessionRepository) Get(ctx context.Context, id string) (*upal.ContentSession, error) {
	s, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("content session %q: %w", id, ErrNotFound)
	}
	return s, err
}

func (r *MemoryContentSessionRepository) List(ctx context.Context) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return !s.IsTemplate && s.ArchivedAt == nil
	})
}

func (r *MemoryContentSessionRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID && !s.IsTemplate && s.ArchivedAt == nil
	})
}

func (r *MemoryContentSessionRepository) ListByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.Status == status && !s.IsTemplate && s.ArchivedAt == nil
	})
}

func (r *MemoryContentSessionRepository) ListAllByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.Status == status && !s.IsTemplate
	})
}

func (r *MemoryContentSessionRepository) ListByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID && s.Status == status && !s.IsTemplate && s.ArchivedAt == nil
	})
}

func (r *MemoryContentSessionRepository) ListArchivedByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID && s.ArchivedAt != nil
	})
}

func (r *MemoryContentSessionRepository) ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID && s.IsTemplate && s.ArchivedAt == nil
	})
}

func (r *MemoryContentSessionRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return fmt.Errorf("content session %q: %w", id, ErrNotFound)
	}
	return err
}

func (r *MemoryContentSessionRepository) Update(ctx context.Context, s *upal.ContentSession) error {
	if !r.store.Has(ctx, s.ID) {
		return fmt.Errorf("content session %q: %w", s.ID, ErrNotFound)
	}
	return r.store.Set(ctx, s)
}

// --- SourceFetch ---

type MemorySourceFetchRepository struct {
	store *memstore.Store[*upal.SourceFetch]
}

func NewMemorySourceFetchRepository() *MemorySourceFetchRepository {
	return &MemorySourceFetchRepository{
		store: memstore.New(func(sf *upal.SourceFetch) string { return sf.ID }),
	}
}

func (r *MemorySourceFetchRepository) Create(ctx context.Context, sf *upal.SourceFetch) error {
	return r.store.Set(ctx, sf)
}

func (r *MemorySourceFetchRepository) Update(ctx context.Context, sf *upal.SourceFetch) error {
	return r.store.Set(ctx, sf)
}

func (r *MemorySourceFetchRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	return r.store.Filter(ctx, func(sf *upal.SourceFetch) bool {
		return sf.SessionID == sessionID
	})
}

// --- LLMAnalysis ---

type MemoryLLMAnalysisRepository struct {
	store *memstore.Store[*upal.LLMAnalysis]
}

func NewMemoryLLMAnalysisRepository() *MemoryLLMAnalysisRepository {
	return &MemoryLLMAnalysisRepository{
		store: memstore.New(func(a *upal.LLMAnalysis) string { return a.ID }),
	}
}

func (r *MemoryLLMAnalysisRepository) Create(ctx context.Context, a *upal.LLMAnalysis) error {
	return r.store.Set(ctx, a)
}

func (r *MemoryLLMAnalysisRepository) GetBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error) {
	all, err := r.store.Filter(ctx, func(a *upal.LLMAnalysis) bool {
		return a.SessionID == sessionID
	})
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("llm analysis for session %q: %w", sessionID, ErrNotFound)
	}
	// Return most recently created (deterministic: latest CreatedAt wins).
	latest := all[0]
	for _, a := range all[1:] {
		if a.CreatedAt.After(latest.CreatedAt) {
			latest = a
		}
	}
	return latest, nil
}

func (r *MemoryLLMAnalysisRepository) Update(ctx context.Context, a *upal.LLMAnalysis) error {
	if !r.store.Has(ctx, a.ID) {
		return fmt.Errorf("llm analysis %q: %w", a.ID, ErrNotFound)
	}
	return r.store.Set(ctx, a)
}

// --- PublishedContent ---

type MemoryPublishedContentRepository struct {
	store *memstore.Store[*upal.PublishedContent]
}

func NewMemoryPublishedContentRepository() *MemoryPublishedContentRepository {
	return &MemoryPublishedContentRepository{
		store: memstore.New(func(pc *upal.PublishedContent) string { return pc.ID }),
	}
}

func (r *MemoryPublishedContentRepository) Create(ctx context.Context, pc *upal.PublishedContent) error {
	return r.store.Set(ctx, pc)
}

func (r *MemoryPublishedContentRepository) List(ctx context.Context) ([]*upal.PublishedContent, error) {
	return r.store.All(ctx)
}

func (r *MemoryPublishedContentRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error) {
	return r.store.Filter(ctx, func(pc *upal.PublishedContent) bool {
		return pc.SessionID == sessionID
	})
}

func (r *MemoryPublishedContentRepository) ListByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error) {
	return r.store.Filter(ctx, func(pc *upal.PublishedContent) bool {
		return pc.Channel == channel
	})
}

func (r *MemoryPublishedContentRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	all, err := r.store.All(ctx)
	if err != nil {
		return err
	}
	for _, pc := range all {
		if pc.SessionID == sessionID {
			if err := r.store.Delete(ctx, pc.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// --- SurgeEvent ---

type MemorySurgeEventRepository struct {
	store *memstore.Store[*upal.SurgeEvent]
}

func NewMemorySurgeEventRepository() *MemorySurgeEventRepository {
	return &MemorySurgeEventRepository{
		store: memstore.New(func(se *upal.SurgeEvent) string { return se.ID }),
	}
}

func (r *MemorySurgeEventRepository) Create(ctx context.Context, se *upal.SurgeEvent) error {
	return r.store.Set(ctx, se)
}

func (r *MemorySurgeEventRepository) List(ctx context.Context) ([]*upal.SurgeEvent, error) {
	return r.store.All(ctx)
}

func (r *MemorySurgeEventRepository) ListActive(ctx context.Context) ([]*upal.SurgeEvent, error) {
	return r.store.Filter(ctx, func(se *upal.SurgeEvent) bool {
		return !se.Dismissed
	})
}

func (r *MemorySurgeEventRepository) Get(ctx context.Context, id string) (*upal.SurgeEvent, error) {
	se, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("surge event %q: %w", id, ErrNotFound)
	}
	return se, err
}

func (r *MemorySurgeEventRepository) Update(ctx context.Context, se *upal.SurgeEvent) error {
	if !r.store.Has(ctx, se.ID) {
		return fmt.Errorf("surge event %q: %w", se.ID, ErrNotFound)
	}
	return r.store.Set(ctx, se)
}

// --- WorkflowResult ---

type MemoryWorkflowResultRepository struct {
	mu      sync.RWMutex
	results map[string][]upal.WorkflowResult // sessionID → results
}

func NewMemoryWorkflowResultRepository() *MemoryWorkflowResultRepository {
	return &MemoryWorkflowResultRepository{
		results: make(map[string][]upal.WorkflowResult),
	}
}

func (r *MemoryWorkflowResultRepository) Save(_ context.Context, sessionID string, results []upal.WorkflowResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]upal.WorkflowResult, len(results))
	copy(cp, results)
	r.results[sessionID] = cp
	return nil
}

func (r *MemoryWorkflowResultRepository) GetBySession(_ context.Context, sessionID string) ([]upal.WorkflowResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	orig, ok := r.results[sessionID]
	if !ok {
		return []upal.WorkflowResult{}, nil
	}
	cp := make([]upal.WorkflowResult, len(orig))
	copy(cp, orig)
	return cp, nil
}

func (r *MemoryWorkflowResultRepository) DeleteBySession(_ context.Context, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.results, sessionID)
	return nil
}
