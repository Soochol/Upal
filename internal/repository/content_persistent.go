package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/soochol/upal/internal/upal"
)

// ContentDB defines the DB-layer methods needed by the persistent content repos.
// *db.DB satisfies this interface.
type ContentDB interface {
	CreateContentSession(ctx context.Context, s *upal.ContentSession) error
	GetContentSession(ctx context.Context, id string) (*upal.ContentSession, error)
	ListContentSessions(ctx context.Context) ([]*upal.ContentSession, error)
	ListContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	ListContentSessionsByStatus(ctx context.Context, status string) ([]*upal.ContentSession, error)
	ListAllContentSessionsByStatus(ctx context.Context, status string) ([]*upal.ContentSession, error)
	ListContentSessionsByPipelineAndStatus(ctx context.Context, pipelineID, status string) ([]*upal.ContentSession, error)
	UpdateContentSession(ctx context.Context, s *upal.ContentSession) error
	ListArchivedContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	ListTemplateContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	DeleteContentSession(ctx context.Context, id string) error
	CreateSourceFetch(ctx context.Context, sf *upal.SourceFetch) error
	ListSourceFetchesBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error)
	CreateLLMAnalysis(ctx context.Context, a *upal.LLMAnalysis) error
	GetLLMAnalysisBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error)
	UpdateLLMAnalysis(ctx context.Context, a *upal.LLMAnalysis) error
	CreatePublishedContent(ctx context.Context, pc *upal.PublishedContent) error
	ListPublishedContent(ctx context.Context) ([]*upal.PublishedContent, error)
	ListPublishedContentBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error)
	ListPublishedContentByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error)
	DeletePublishedContentBySession(ctx context.Context, sessionID string) error
	CreateSurgeEvent(ctx context.Context, se *upal.SurgeEvent) error
	GetSurgeEvent(ctx context.Context, id string) (*upal.SurgeEvent, error)
	ListSurgeEvents(ctx context.Context) ([]*upal.SurgeEvent, error)
	ListActiveSurgeEvents(ctx context.Context) ([]*upal.SurgeEvent, error)
	UpdateSurgeEvent(ctx context.Context, se *upal.SurgeEvent) error
	SaveWorkflowResults(ctx context.Context, sessionID string, results []upal.WorkflowResult) error
	GetWorkflowResultsBySession(ctx context.Context, sessionID string) ([]upal.WorkflowResult, error)
	DeleteWorkflowResultsBySession(ctx context.Context, sessionID string) error
}

// PersistentContentSessionRepository wraps MemoryContentSessionRepository with DB backend.
type PersistentContentSessionRepository struct {
	mem *MemoryContentSessionRepository
	db  ContentDB
}

func NewPersistentContentSessionRepository(mem *MemoryContentSessionRepository, db ContentDB) *PersistentContentSessionRepository {
	return &PersistentContentSessionRepository{mem: mem, db: db}
}

func (r *PersistentContentSessionRepository) Create(ctx context.Context, s *upal.ContentSession) error {
	_ = r.mem.Create(ctx, s)
	if err := r.db.CreateContentSession(ctx, s); err != nil {
		return fmt.Errorf("db create content_session: %w", err)
	}
	return nil
}

func (r *PersistentContentSessionRepository) Get(ctx context.Context, id string) (*upal.ContentSession, error) {
	if s, err := r.mem.Get(ctx, id); err == nil {
		return s, nil
	}
	s, err := r.db.GetContentSession(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, s)
	return s, nil
}

func (r *PersistentContentSessionRepository) List(ctx context.Context) ([]*upal.ContentSession, error) {
	sessions, err := r.db.ListContentSessions(ctx)
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list content_sessions failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentContentSessionRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	sessions, err := r.db.ListContentSessionsByPipeline(ctx, pipelineID)
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list content_sessions by pipeline failed, falling back to in-memory", "err", err)
	return r.mem.ListByPipeline(ctx, pipelineID)
}

func (r *PersistentContentSessionRepository) ListByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	sessions, err := r.db.ListContentSessionsByStatus(ctx, string(status))
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list content_sessions by status failed, falling back to in-memory", "err", err)
	return r.mem.ListByStatus(ctx, status)
}

func (r *PersistentContentSessionRepository) ListAllByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	sessions, err := r.db.ListAllContentSessionsByStatus(ctx, string(status))
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list all content_sessions by status failed, falling back to in-memory", "err", err)
	return r.mem.ListAllByStatus(ctx, status)
}

func (r *PersistentContentSessionRepository) ListByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	sessions, err := r.db.ListContentSessionsByPipelineAndStatus(ctx, pipelineID, string(status))
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list content_sessions by pipeline+status failed, falling back to in-memory", "err", err)
	return r.mem.ListByPipelineAndStatus(ctx, pipelineID, status)
}

func (r *PersistentContentSessionRepository) ListArchivedByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	sessions, err := r.db.ListArchivedContentSessionsByPipeline(ctx, pipelineID)
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list archived content_sessions failed, falling back to in-memory", "err", err)
	return r.mem.ListArchivedByPipeline(ctx, pipelineID)
}

func (r *PersistentContentSessionRepository) ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	sessions, err := r.db.ListTemplateContentSessionsByPipeline(ctx, pipelineID)
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list template content_sessions failed, falling back to in-memory", "err", err)
	return r.mem.ListTemplatesByPipeline(ctx, pipelineID)
}

func (r *PersistentContentSessionRepository) Update(ctx context.Context, s *upal.ContentSession) error {
	_ = r.mem.Update(ctx, s)
	if err := r.db.UpdateContentSession(ctx, s); err != nil {
		return fmt.Errorf("db update content_session: %w", err)
	}
	return nil
}

func (r *PersistentContentSessionRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	if err := r.db.DeleteContentSession(ctx, id); err != nil {
		return fmt.Errorf("db delete content_session: %w", err)
	}
	return nil
}

// PersistentSourceFetchRepository wraps MemorySourceFetchRepository with DB backend.
type PersistentSourceFetchRepository struct {
	mem *MemorySourceFetchRepository
	db  ContentDB
}

func NewPersistentSourceFetchRepository(mem *MemorySourceFetchRepository, db ContentDB) *PersistentSourceFetchRepository {
	return &PersistentSourceFetchRepository{mem: mem, db: db}
}

func (r *PersistentSourceFetchRepository) Create(ctx context.Context, sf *upal.SourceFetch) error {
	_ = r.mem.Create(ctx, sf)
	if err := r.db.CreateSourceFetch(ctx, sf); err != nil {
		return fmt.Errorf("db create source_fetch: %w", err)
	}
	return nil
}

func (r *PersistentSourceFetchRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	fetches, err := r.db.ListSourceFetchesBySession(ctx, sessionID)
	if err == nil {
		return fetches, nil
	}
	slog.Warn("db list source_fetches failed, falling back to in-memory", "err", err)
	return r.mem.ListBySession(ctx, sessionID)
}

// PersistentLLMAnalysisRepository wraps MemoryLLMAnalysisRepository with DB backend.
type PersistentLLMAnalysisRepository struct {
	mem *MemoryLLMAnalysisRepository
	db  ContentDB
}

func NewPersistentLLMAnalysisRepository(mem *MemoryLLMAnalysisRepository, db ContentDB) *PersistentLLMAnalysisRepository {
	return &PersistentLLMAnalysisRepository{mem: mem, db: db}
}

func (r *PersistentLLMAnalysisRepository) Create(ctx context.Context, a *upal.LLMAnalysis) error {
	_ = r.mem.Create(ctx, a)
	if err := r.db.CreateLLMAnalysis(ctx, a); err != nil {
		return fmt.Errorf("db create llm_analysis: %w", err)
	}
	return nil
}

func (r *PersistentLLMAnalysisRepository) GetBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error) {
	if a, err := r.mem.GetBySession(ctx, sessionID); err == nil {
		return a, nil
	}
	return r.db.GetLLMAnalysisBySession(ctx, sessionID)
}

func (r *PersistentLLMAnalysisRepository) Update(ctx context.Context, a *upal.LLMAnalysis) error {
	_ = r.mem.Update(ctx, a)
	if err := r.db.UpdateLLMAnalysis(ctx, a); err != nil {
		return fmt.Errorf("db update llm_analysis: %w", err)
	}
	return nil
}

// PersistentPublishedContentRepository wraps MemoryPublishedContentRepository with DB backend.
type PersistentPublishedContentRepository struct {
	mem *MemoryPublishedContentRepository
	db  ContentDB
}

func NewPersistentPublishedContentRepository(mem *MemoryPublishedContentRepository, db ContentDB) *PersistentPublishedContentRepository {
	return &PersistentPublishedContentRepository{mem: mem, db: db}
}

func (r *PersistentPublishedContentRepository) Create(ctx context.Context, pc *upal.PublishedContent) error {
	_ = r.mem.Create(ctx, pc)
	if err := r.db.CreatePublishedContent(ctx, pc); err != nil {
		return fmt.Errorf("db create published_content: %w", err)
	}
	return nil
}

func (r *PersistentPublishedContentRepository) List(ctx context.Context) ([]*upal.PublishedContent, error) {
	pcs, err := r.db.ListPublishedContent(ctx)
	if err == nil {
		return pcs, nil
	}
	slog.Warn("db list published_content failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentPublishedContentRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error) {
	pcs, err := r.db.ListPublishedContentBySession(ctx, sessionID)
	if err == nil {
		return pcs, nil
	}
	slog.Warn("db list published_content by session failed, falling back to in-memory", "err", err)
	return r.mem.ListBySession(ctx, sessionID)
}

func (r *PersistentPublishedContentRepository) ListByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error) {
	pcs, err := r.db.ListPublishedContentByChannel(ctx, channel)
	if err == nil {
		return pcs, nil
	}
	slog.Warn("db list published_content by channel failed, falling back to in-memory", "err", err)
	return r.mem.ListByChannel(ctx, channel)
}

func (r *PersistentPublishedContentRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	_ = r.mem.DeleteBySession(ctx, sessionID)
	if err := r.db.DeletePublishedContentBySession(ctx, sessionID); err != nil {
		return fmt.Errorf("db delete published_content by session: %w", err)
	}
	return nil
}

// PersistentSurgeEventRepository wraps MemorySurgeEventRepository with DB backend.
type PersistentSurgeEventRepository struct {
	mem *MemorySurgeEventRepository
	db  ContentDB
}

func NewPersistentSurgeEventRepository(mem *MemorySurgeEventRepository, db ContentDB) *PersistentSurgeEventRepository {
	return &PersistentSurgeEventRepository{mem: mem, db: db}
}

func (r *PersistentSurgeEventRepository) Create(ctx context.Context, se *upal.SurgeEvent) error {
	_ = r.mem.Create(ctx, se)
	if err := r.db.CreateSurgeEvent(ctx, se); err != nil {
		return fmt.Errorf("db create surge_event: %w", err)
	}
	return nil
}

func (r *PersistentSurgeEventRepository) List(ctx context.Context) ([]*upal.SurgeEvent, error) {
	events, err := r.db.ListSurgeEvents(ctx)
	if err == nil {
		return events, nil
	}
	slog.Warn("db list surge_events failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentSurgeEventRepository) ListActive(ctx context.Context) ([]*upal.SurgeEvent, error) {
	events, err := r.db.ListActiveSurgeEvents(ctx)
	if err == nil {
		return events, nil
	}
	slog.Warn("db list active surge_events failed, falling back to in-memory", "err", err)
	return r.mem.ListActive(ctx)
}

func (r *PersistentSurgeEventRepository) Get(ctx context.Context, id string) (*upal.SurgeEvent, error) {
	if se, err := r.mem.Get(ctx, id); err == nil {
		return se, nil
	}
	se, err := r.db.GetSurgeEvent(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, se)
	return se, nil
}

func (r *PersistentSurgeEventRepository) Update(ctx context.Context, se *upal.SurgeEvent) error {
	_ = r.mem.Update(ctx, se)
	if err := r.db.UpdateSurgeEvent(ctx, se); err != nil {
		return fmt.Errorf("db update surge_event: %w", err)
	}
	return nil
}

// PersistentWorkflowResultRepository wraps MemoryWorkflowResultRepository with DB backend.
type PersistentWorkflowResultRepository struct {
	mem *MemoryWorkflowResultRepository
	db  ContentDB
}

func NewPersistentWorkflowResultRepository(mem *MemoryWorkflowResultRepository, db ContentDB) *PersistentWorkflowResultRepository {
	return &PersistentWorkflowResultRepository{mem: mem, db: db}
}

func (r *PersistentWorkflowResultRepository) Save(ctx context.Context, sessionID string, results []upal.WorkflowResult) error {
	_ = r.mem.Save(ctx, sessionID, results)
	if err := r.db.SaveWorkflowResults(ctx, sessionID, results); err != nil {
		return fmt.Errorf("db save workflow_results: %w", err)
	}
	return nil
}

func (r *PersistentWorkflowResultRepository) GetBySession(ctx context.Context, sessionID string) ([]upal.WorkflowResult, error) {
	if results, err := r.mem.GetBySession(ctx, sessionID); err == nil && len(results) > 0 {
		return results, nil
	}
	results, err := r.db.GetWorkflowResultsBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 {
		_ = r.mem.Save(ctx, sessionID, results)
	}
	return results, nil
}

func (r *PersistentWorkflowResultRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	_ = r.mem.DeleteBySession(ctx, sessionID)
	if err := r.db.DeleteWorkflowResultsBySession(ctx, sessionID); err != nil {
		return fmt.Errorf("db delete workflow_results: %w", err)
	}
	return nil
}
