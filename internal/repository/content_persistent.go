package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/soochol/upal/internal/upal"
)

// ContentDB defines the database methods used by persistent content repositories.
type ContentDB interface {
	CreateContentSession(ctx context.Context, userID string, s *upal.ContentSession) error
	GetContentSession(ctx context.Context, userID string, id string) (*upal.ContentSession, error)
	ListContentSessions(ctx context.Context, userID string) ([]*upal.ContentSession, error)
	ListContentSessionsByPipeline(ctx context.Context, userID string, pipelineID string) ([]*upal.ContentSession, error)
	ListContentSessionsByStatus(ctx context.Context, userID string, status string) ([]*upal.ContentSession, error)
	ListAllContentSessionsByStatus(ctx context.Context, userID string, status string) ([]*upal.ContentSession, error)
	ListContentSessionsByPipelineAndStatus(ctx context.Context, userID string, pipelineID, status string) ([]*upal.ContentSession, error)
	UpdateContentSession(ctx context.Context, userID string, s *upal.ContentSession) error
	DeleteContentSession(ctx context.Context, userID string, id string) error
	CreateSourceFetch(ctx context.Context, userID string, sf *upal.SourceFetch) error
	ListSourceFetchesBySession(ctx context.Context, userID string, sessionID string) ([]*upal.SourceFetch, error)
	DeleteSourceFetchesBySession(ctx context.Context, userID string, sessionID string) error
	CreateLLMAnalysis(ctx context.Context, userID string, a *upal.LLMAnalysis) error
	GetLLMAnalysisBySession(ctx context.Context, userID string, sessionID string) (*upal.LLMAnalysis, error)
	UpdateLLMAnalysis(ctx context.Context, userID string, a *upal.LLMAnalysis) error
	DeleteLLMAnalysesBySession(ctx context.Context, userID string, sessionID string) error
	CreatePublishedContent(ctx context.Context, userID string, pc *upal.PublishedContent) error
	ListPublishedContent(ctx context.Context, userID string) ([]*upal.PublishedContent, error)
	ListPublishedContentBySession(ctx context.Context, userID string, sessionID string) ([]*upal.PublishedContent, error)
	ListPublishedContentByChannel(ctx context.Context, userID string, channel string) ([]*upal.PublishedContent, error)
	DeletePublishedContentBySession(ctx context.Context, userID string, sessionID string) error
	CreateSurgeEvent(ctx context.Context, userID string, se *upal.SurgeEvent) error
	GetSurgeEvent(ctx context.Context, userID string, id string) (*upal.SurgeEvent, error)
	ListSurgeEvents(ctx context.Context, userID string) ([]*upal.SurgeEvent, error)
	ListActiveSurgeEvents(ctx context.Context, userID string) ([]*upal.SurgeEvent, error)
	UpdateSurgeEvent(ctx context.Context, userID string, se *upal.SurgeEvent) error
	SaveWorkflowResults(ctx context.Context, userID string, sessionID string, results []upal.WorkflowResult) error
	GetWorkflowResultsBySession(ctx context.Context, userID string, sessionID string) ([]upal.WorkflowResult, error)
	DeleteWorkflowResultsBySession(ctx context.Context, userID string, sessionID string) error
}

type PersistentContentSessionRepository struct {
	mem *MemoryContentSessionRepository
	db  ContentDB
}

func NewPersistentContentSessionRepository(mem *MemoryContentSessionRepository, db ContentDB) *PersistentContentSessionRepository {
	return &PersistentContentSessionRepository{mem: mem, db: db}
}

func (r *PersistentContentSessionRepository) Create(ctx context.Context, s *upal.ContentSession) error {
	_ = r.mem.Create(ctx, s)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.CreateContentSession(ctx, userID, s); err != nil {
		return fmt.Errorf("db create content_session: %w", err)
	}
	return nil
}

func (r *PersistentContentSessionRepository) Get(ctx context.Context, id string) (*upal.ContentSession, error) {
	if s, err := r.mem.Get(ctx, id); err == nil {
		return s, nil
	}
	userID := upal.UserIDFromContext(ctx)
	s, err := r.db.GetContentSession(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, s)
	return s, nil
}

func (r *PersistentContentSessionRepository) List(ctx context.Context) ([]*upal.ContentSession, error) {
	userID := upal.UserIDFromContext(ctx)
	sessions, err := r.db.ListContentSessions(ctx, userID)
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list content_sessions failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentContentSessionRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	userID := upal.UserIDFromContext(ctx)
	sessions, err := r.db.ListContentSessionsByPipeline(ctx, userID, pipelineID)
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list content_sessions by pipeline failed, falling back to in-memory", "err", err)
	return r.mem.ListByPipeline(ctx, pipelineID)
}

func (r *PersistentContentSessionRepository) ListByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	userID := upal.UserIDFromContext(ctx)
	sessions, err := r.db.ListContentSessionsByStatus(ctx, userID, string(status))
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list content_sessions by status failed, falling back to in-memory", "err", err)
	return r.mem.ListByStatus(ctx, status)
}

func (r *PersistentContentSessionRepository) ListAllByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return r.ListByStatus(ctx, status)
}

func (r *PersistentContentSessionRepository) ListByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	userID := upal.UserIDFromContext(ctx)
	sessions, err := r.db.ListContentSessionsByPipelineAndStatus(ctx, userID, pipelineID, string(status))
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list content_sessions by pipeline+status failed, falling back to in-memory", "err", err)
	return r.mem.ListByPipelineAndStatus(ctx, pipelineID, status)
}

func (r *PersistentContentSessionRepository) ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.mem.ListTemplatesByPipeline(ctx, pipelineID)
}

func (r *PersistentContentSessionRepository) Update(ctx context.Context, s *upal.ContentSession) error {
	_ = r.mem.Update(ctx, s)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.UpdateContentSession(ctx, userID, s); err != nil {
		return fmt.Errorf("db update content_session: %w", err)
	}
	return nil
}

func (r *PersistentContentSessionRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteContentSession(ctx, userID, id); err != nil {
		return fmt.Errorf("db delete content_session: %w", err)
	}
	return nil
}

type PersistentSourceFetchRepository struct {
	mem *MemorySourceFetchRepository
	db  ContentDB
}

func NewPersistentSourceFetchRepository(mem *MemorySourceFetchRepository, db ContentDB) *PersistentSourceFetchRepository {
	return &PersistentSourceFetchRepository{mem: mem, db: db}
}

func (r *PersistentSourceFetchRepository) Create(ctx context.Context, sf *upal.SourceFetch) error {
	_ = r.mem.Create(ctx, sf)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.CreateSourceFetch(ctx, userID, sf); err != nil {
		return fmt.Errorf("db create source_fetch: %w", err)
	}
	return nil
}

func (r *PersistentSourceFetchRepository) Update(ctx context.Context, sf *upal.SourceFetch) error {
	return r.mem.Update(ctx, sf)
}

func (r *PersistentSourceFetchRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	userID := upal.UserIDFromContext(ctx)
	fetches, err := r.db.ListSourceFetchesBySession(ctx, userID, sessionID)
	if err == nil {
		return fetches, nil
	}
	slog.Warn("db list source_fetches failed, falling back to in-memory", "err", err)
	return r.mem.ListBySession(ctx, sessionID)
}

func (r *PersistentSourceFetchRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	_ = r.mem.DeleteBySession(ctx, sessionID)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteSourceFetchesBySession(ctx, userID, sessionID); err != nil {
		return fmt.Errorf("db delete source_fetches by session: %w", err)
	}
	return nil
}

type PersistentLLMAnalysisRepository struct {
	mem *MemoryLLMAnalysisRepository
	db  ContentDB
}

func NewPersistentLLMAnalysisRepository(mem *MemoryLLMAnalysisRepository, db ContentDB) *PersistentLLMAnalysisRepository {
	return &PersistentLLMAnalysisRepository{mem: mem, db: db}
}

func (r *PersistentLLMAnalysisRepository) Create(ctx context.Context, a *upal.LLMAnalysis) error {
	_ = r.mem.Create(ctx, a)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.CreateLLMAnalysis(ctx, userID, a); err != nil {
		return fmt.Errorf("db create llm_analysis: %w", err)
	}
	return nil
}

func (r *PersistentLLMAnalysisRepository) GetBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error) {
	if a, err := r.mem.GetBySession(ctx, sessionID); err == nil {
		return a, nil
	}
	userID := upal.UserIDFromContext(ctx)
	return r.db.GetLLMAnalysisBySession(ctx, userID, sessionID)
}

func (r *PersistentLLMAnalysisRepository) Update(ctx context.Context, a *upal.LLMAnalysis) error {
	_ = r.mem.Update(ctx, a)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.UpdateLLMAnalysis(ctx, userID, a); err != nil {
		return fmt.Errorf("db update llm_analysis: %w", err)
	}
	return nil
}

func (r *PersistentLLMAnalysisRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	_ = r.mem.DeleteBySession(ctx, sessionID)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteLLMAnalysesBySession(ctx, userID, sessionID); err != nil {
		return fmt.Errorf("db delete llm_analyses by session: %w", err)
	}
	return nil
}

type PersistentPublishedContentRepository struct {
	mem *MemoryPublishedContentRepository
	db  ContentDB
}

func NewPersistentPublishedContentRepository(mem *MemoryPublishedContentRepository, db ContentDB) *PersistentPublishedContentRepository {
	return &PersistentPublishedContentRepository{mem: mem, db: db}
}

func (r *PersistentPublishedContentRepository) Create(ctx context.Context, pc *upal.PublishedContent) error {
	_ = r.mem.Create(ctx, pc)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.CreatePublishedContent(ctx, userID, pc); err != nil {
		return fmt.Errorf("db create published_content: %w", err)
	}
	return nil
}

func (r *PersistentPublishedContentRepository) List(ctx context.Context) ([]*upal.PublishedContent, error) {
	userID := upal.UserIDFromContext(ctx)
	pcs, err := r.db.ListPublishedContent(ctx, userID)
	if err == nil {
		return pcs, nil
	}
	slog.Warn("db list published_content failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentPublishedContentRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error) {
	userID := upal.UserIDFromContext(ctx)
	pcs, err := r.db.ListPublishedContentBySession(ctx, userID, sessionID)
	if err == nil {
		return pcs, nil
	}
	slog.Warn("db list published_content by session failed, falling back to in-memory", "err", err)
	return r.mem.ListBySession(ctx, sessionID)
}

func (r *PersistentPublishedContentRepository) ListByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error) {
	userID := upal.UserIDFromContext(ctx)
	pcs, err := r.db.ListPublishedContentByChannel(ctx, userID, channel)
	if err == nil {
		return pcs, nil
	}
	slog.Warn("db list published_content by channel failed, falling back to in-memory", "err", err)
	return r.mem.ListByChannel(ctx, channel)
}

func (r *PersistentPublishedContentRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	_ = r.mem.DeleteBySession(ctx, sessionID)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeletePublishedContentBySession(ctx, userID, sessionID); err != nil {
		return fmt.Errorf("db delete published_content by session: %w", err)
	}
	return nil
}

type PersistentSurgeEventRepository struct {
	mem *MemorySurgeEventRepository
	db  ContentDB
}

func NewPersistentSurgeEventRepository(mem *MemorySurgeEventRepository, db ContentDB) *PersistentSurgeEventRepository {
	return &PersistentSurgeEventRepository{mem: mem, db: db}
}

func (r *PersistentSurgeEventRepository) Create(ctx context.Context, se *upal.SurgeEvent) error {
	_ = r.mem.Create(ctx, se)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.CreateSurgeEvent(ctx, userID, se); err != nil {
		return fmt.Errorf("db create surge_event: %w", err)
	}
	return nil
}

func (r *PersistentSurgeEventRepository) List(ctx context.Context) ([]*upal.SurgeEvent, error) {
	userID := upal.UserIDFromContext(ctx)
	events, err := r.db.ListSurgeEvents(ctx, userID)
	if err == nil {
		return events, nil
	}
	slog.Warn("db list surge_events failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentSurgeEventRepository) ListActive(ctx context.Context) ([]*upal.SurgeEvent, error) {
	userID := upal.UserIDFromContext(ctx)
	events, err := r.db.ListActiveSurgeEvents(ctx, userID)
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
	userID := upal.UserIDFromContext(ctx)
	se, err := r.db.GetSurgeEvent(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, se)
	return se, nil
}

func (r *PersistentSurgeEventRepository) Update(ctx context.Context, se *upal.SurgeEvent) error {
	_ = r.mem.Update(ctx, se)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.UpdateSurgeEvent(ctx, userID, se); err != nil {
		return fmt.Errorf("db update surge_event: %w", err)
	}
	return nil
}

type PersistentWorkflowResultRepository struct {
	mem *MemoryWorkflowResultRepository
	db  ContentDB
}

func NewPersistentWorkflowResultRepository(mem *MemoryWorkflowResultRepository, db ContentDB) *PersistentWorkflowResultRepository {
	return &PersistentWorkflowResultRepository{mem: mem, db: db}
}

func (r *PersistentWorkflowResultRepository) Save(ctx context.Context, sessionID string, results []upal.WorkflowResult) error {
	_ = r.mem.Save(ctx, sessionID, results)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.SaveWorkflowResults(ctx, userID, sessionID, results); err != nil {
		return fmt.Errorf("db save workflow_results: %w", err)
	}
	return nil
}

func (r *PersistentWorkflowResultRepository) GetBySession(ctx context.Context, sessionID string) ([]upal.WorkflowResult, error) {
	if results, err := r.mem.GetBySession(ctx, sessionID); err == nil && len(results) > 0 {
		return results, nil
	}
	userID := upal.UserIDFromContext(ctx)
	results, err := r.db.GetWorkflowResultsBySession(ctx, userID, sessionID)
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
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteWorkflowResultsBySession(ctx, userID, sessionID); err != nil {
		return fmt.Errorf("db delete workflow_results: %w", err)
	}
	return nil
}
