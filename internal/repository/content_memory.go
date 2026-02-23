package repository

import (
	"context"
	"errors"
	"fmt"

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
		return nil, fmt.Errorf("content session %q not found", id)
	}
	return s, err
}

func (r *MemoryContentSessionRepository) List(ctx context.Context) ([]*upal.ContentSession, error) {
	return r.store.All(ctx)
}

func (r *MemoryContentSessionRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID
	})
}

func (r *MemoryContentSessionRepository) Update(ctx context.Context, s *upal.ContentSession) error {
	if !r.store.Has(ctx, s.ID) {
		return fmt.Errorf("content session %q not found", s.ID)
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
		return nil, fmt.Errorf("llm analysis for session %q not found", sessionID)
	}
	return all[len(all)-1], nil
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
		return nil, fmt.Errorf("surge event %q not found", id)
	}
	return se, err
}

func (r *MemorySurgeEventRepository) Update(ctx context.Context, se *upal.SurgeEvent) error {
	if !r.store.Has(ctx, se.ID) {
		return fmt.Errorf("surge event %q not found", se.ID)
	}
	return r.store.Set(ctx, se)
}
