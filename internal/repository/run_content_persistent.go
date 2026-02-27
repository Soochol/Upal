package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/soochol/upal/internal/upal"
)

// RunContentDB defines the database methods used by V2 (Session/Run) persistent repositories.
type RunContentDB interface {
	CreateRunSourceFetch(ctx context.Context, userID string, sf *upal.SourceFetch) error
	ListRunSourceFetches(ctx context.Context, userID string, runID string) ([]*upal.SourceFetch, error)
	DeleteRunSourceFetches(ctx context.Context, userID string, runID string) error
	CreateRunLLMAnalysis(ctx context.Context, userID string, a *upal.LLMAnalysis) error
	GetRunLLMAnalysis(ctx context.Context, userID string, runID string) (*upal.LLMAnalysis, error)
	UpdateRunLLMAnalysis(ctx context.Context, userID string, a *upal.LLMAnalysis) error
	DeleteRunLLMAnalyses(ctx context.Context, userID string, runID string) error
}

// PersistentRunSourceFetchRepository stores source fetches in upal_source_fetches (keyed by run_id).
type PersistentRunSourceFetchRepository struct {
	mem *MemorySourceFetchRepository
	db  RunContentDB
}

func NewPersistentRunSourceFetchRepository(mem *MemorySourceFetchRepository, db RunContentDB) *PersistentRunSourceFetchRepository {
	return &PersistentRunSourceFetchRepository{mem: mem, db: db}
}

func (r *PersistentRunSourceFetchRepository) Create(ctx context.Context, sf *upal.SourceFetch) error {
	_ = r.mem.Create(ctx, sf)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.CreateRunSourceFetch(ctx, userID, sf); err != nil {
		return fmt.Errorf("db create run_source_fetch: %w", err)
	}
	return nil
}

func (r *PersistentRunSourceFetchRepository) Update(ctx context.Context, sf *upal.SourceFetch) error {
	return r.mem.Update(ctx, sf)
}

func (r *PersistentRunSourceFetchRepository) ListBySession(ctx context.Context, runID string) ([]*upal.SourceFetch, error) {
	userID := upal.UserIDFromContext(ctx)
	fetches, err := r.db.ListRunSourceFetches(ctx, userID, runID)
	if err == nil {
		return fetches, nil
	}
	slog.Warn("db list run_source_fetches failed, falling back to in-memory", "err", err)
	return r.mem.ListBySession(ctx, runID)
}

func (r *PersistentRunSourceFetchRepository) DeleteBySession(ctx context.Context, runID string) error {
	_ = r.mem.DeleteBySession(ctx, runID)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteRunSourceFetches(ctx, userID, runID); err != nil {
		return fmt.Errorf("db delete run_source_fetches: %w", err)
	}
	return nil
}

// PersistentRunLLMAnalysisRepository stores analyses in upal_llm_analyses (keyed by run_id).
type PersistentRunLLMAnalysisRepository struct {
	mem *MemoryLLMAnalysisRepository
	db  RunContentDB
}

func NewPersistentRunLLMAnalysisRepository(mem *MemoryLLMAnalysisRepository, db RunContentDB) *PersistentRunLLMAnalysisRepository {
	return &PersistentRunLLMAnalysisRepository{mem: mem, db: db}
}

func (r *PersistentRunLLMAnalysisRepository) Create(ctx context.Context, a *upal.LLMAnalysis) error {
	_ = r.mem.Create(ctx, a)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.CreateRunLLMAnalysis(ctx, userID, a); err != nil {
		return fmt.Errorf("db create run_llm_analysis: %w", err)
	}
	return nil
}

func (r *PersistentRunLLMAnalysisRepository) GetBySession(ctx context.Context, runID string) (*upal.LLMAnalysis, error) {
	if a, err := r.mem.GetBySession(ctx, runID); err == nil {
		return a, nil
	}
	userID := upal.UserIDFromContext(ctx)
	return r.db.GetRunLLMAnalysis(ctx, userID, runID)
}

func (r *PersistentRunLLMAnalysisRepository) Update(ctx context.Context, a *upal.LLMAnalysis) error {
	_ = r.mem.Update(ctx, a)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.UpdateRunLLMAnalysis(ctx, userID, a); err != nil {
		return fmt.Errorf("db update run_llm_analysis: %w", err)
	}
	return nil
}

func (r *PersistentRunLLMAnalysisRepository) DeleteBySession(ctx context.Context, runID string) error {
	_ = r.mem.DeleteBySession(ctx, runID)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteRunLLMAnalyses(ctx, userID, runID); err != nil {
		return fmt.Errorf("db delete run_llm_analyses: %w", err)
	}
	return nil
}
