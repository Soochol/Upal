package services

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// ContentSessionService manages content collection sessions and related records.
type ContentSessionService struct {
	sessions  repository.ContentSessionRepository
	fetches   repository.SourceFetchRepository
	analyses  repository.LLMAnalysisRepository
	published repository.PublishedContentRepository
	surges    repository.SurgeEventRepository

	mu              sync.Mutex
	workflowResults map[string][]upal.WorkflowResult // sessionID → results
}

func NewContentSessionService(
	sessions repository.ContentSessionRepository,
	fetches repository.SourceFetchRepository,
	analyses repository.LLMAnalysisRepository,
	published repository.PublishedContentRepository,
	surges repository.SurgeEventRepository,
) *ContentSessionService {
	return &ContentSessionService{
		sessions:        sessions,
		fetches:         fetches,
		analyses:        analyses,
		published:       published,
		surges:          surges,
		workflowResults: make(map[string][]upal.WorkflowResult),
	}
}

// --- ContentSession ---

func (s *ContentSessionService) CreateSession(ctx context.Context, sess *upal.ContentSession) error {
	if sess.PipelineID == "" {
		return fmt.Errorf("pipeline_id is required")
	}
	if sess.ID == "" {
		sess.ID = upal.GenerateID("csess")
	}
	if sess.Status == "" {
		sess.Status = upal.SessionCollecting
	}
	if sess.TriggerType == "" {
		sess.TriggerType = "manual"
	}
	sess.CreatedAt = time.Now()
	return s.sessions.Create(ctx, sess)
}

func (s *ContentSessionService) GetSession(ctx context.Context, id string) (*upal.ContentSession, error) {
	return s.sessions.Get(ctx, id)
}

func (s *ContentSessionService) ListSessions(ctx context.Context) ([]*upal.ContentSession, error) {
	return s.sessions.List(ctx)
}

func (s *ContentSessionService) ListSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return s.sessions.ListByPipeline(ctx, pipelineID)
}

func (s *ContentSessionService) ListSessionsByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return s.sessions.ListByStatus(ctx, status)
}

func (s *ContentSessionService) ListSessionsByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return s.sessions.ListByPipelineAndStatus(ctx, pipelineID, status)
}

func (s *ContentSessionService) UpdateSessionStatus(ctx context.Context, id string, status upal.ContentSessionStatus) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	sess.Status = status
	return s.sessions.Update(ctx, sess)
}

// UpdateSessionSourceCount sets the SourceCount field on a session and persists it.
func (s *ContentSessionService) UpdateSessionSourceCount(ctx context.Context, id string, count int) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	sess.SourceCount = count
	return s.sessions.Update(ctx, sess)
}

func (s *ContentSessionService) ApproveSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now()
	sess.Status = upal.SessionApproved
	sess.ReviewedAt = &now
	return s.sessions.Update(ctx, sess)
}

func (s *ContentSessionService) RejectSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now()
	sess.Status = upal.SessionRejected
	sess.ReviewedAt = &now
	return s.sessions.Update(ctx, sess)
}

// --- SourceFetch ---

func (s *ContentSessionService) RecordSourceFetch(ctx context.Context, sf *upal.SourceFetch) error {
	if sf.ID == "" {
		sf.ID = upal.GenerateID("sfetch")
	}
	sf.FetchedAt = time.Now()
	return s.fetches.Create(ctx, sf)
}

func (s *ContentSessionService) ListSourceFetches(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	return s.fetches.ListBySession(ctx, sessionID)
}

// --- LLMAnalysis ---

func (s *ContentSessionService) RecordAnalysis(ctx context.Context, a *upal.LLMAnalysis) error {
	if a.ID == "" {
		a.ID = upal.GenerateID("anlys")
	}
	a.CreatedAt = time.Now()
	return s.analyses.Create(ctx, a)
}

func (s *ContentSessionService) GetAnalysis(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error) {
	return s.analyses.GetBySession(ctx, sessionID)
}

// UpdateAnalysis updates the summary and insights of an existing analysis.
func (s *ContentSessionService) UpdateAnalysis(ctx context.Context, sessionID string, summary string, insights []string) error {
	analysis, err := s.analyses.GetBySession(ctx, sessionID)
	if err != nil {
		return err
	}
	if analysis == nil {
		return fmt.Errorf("no analysis found for session %s", sessionID)
	}
	analysis.Summary = summary
	analysis.Insights = insights
	return s.analyses.Update(ctx, analysis)
}

// --- PublishedContent ---

func (s *ContentSessionService) RecordPublished(ctx context.Context, pc *upal.PublishedContent) error {
	if pc.ID == "" {
		pc.ID = upal.GenerateID("pub")
	}
	pc.PublishedAt = time.Now()
	return s.published.Create(ctx, pc)
}

func (s *ContentSessionService) ListPublished(ctx context.Context) ([]*upal.PublishedContent, error) {
	return s.published.List(ctx)
}

func (s *ContentSessionService) ListPublishedBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error) {
	return s.published.ListBySession(ctx, sessionID)
}

func (s *ContentSessionService) ListPublishedByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error) {
	return s.published.ListByChannel(ctx, channel)
}

// --- SurgeEvent ---

func (s *ContentSessionService) CreateSurge(ctx context.Context, se *upal.SurgeEvent) error {
	if se.ID == "" {
		se.ID = upal.GenerateID("surge")
	}
	se.CreatedAt = time.Now()
	return s.surges.Create(ctx, se)
}

func (s *ContentSessionService) ListSurges(ctx context.Context) ([]*upal.SurgeEvent, error) {
	return s.surges.List(ctx)
}

func (s *ContentSessionService) ListActiveSurges(ctx context.Context) ([]*upal.SurgeEvent, error) {
	return s.surges.ListActive(ctx)
}

func (s *ContentSessionService) DismissSurge(ctx context.Context, id string) error {
	se, err := s.surges.Get(ctx, id)
	if err != nil {
		return err
	}
	se.Dismissed = true
	return s.surges.Update(ctx, se)
}

// --- Archive / Unarchive / Delete ---

func (s *ContentSessionService) ArchiveSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	if sess.ArchivedAt != nil {
		return fmt.Errorf("session %q is already archived", id)
	}
	now := time.Now()
	sess.ArchivedAt = &now
	return s.sessions.Update(ctx, sess)
}

func (s *ContentSessionService) UnarchiveSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	if sess.ArchivedAt == nil {
		return fmt.Errorf("session %q is not archived", id)
	}
	sess.ArchivedAt = nil
	return s.sessions.Update(ctx, sess)
}

func (s *ContentSessionService) DeleteSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	if sess.ArchivedAt == nil {
		return fmt.Errorf("session %q must be archived before deletion", id)
	}

	// Clean up published_content (no FK cascade)
	if err := s.published.DeleteBySession(ctx, id); err != nil {
		return fmt.Errorf("delete published content: %w", err)
	}

	// Delete session (source_fetches + llm_analyses cascade in DB)
	if err := s.sessions.Delete(ctx, id); err != nil {
		return err
	}

	// Clean up in-memory workflow results
	s.mu.Lock()
	delete(s.workflowResults, id)
	s.mu.Unlock()

	return nil
}

func (s *ContentSessionService) ListArchivedByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return s.sessions.ListArchivedByPipeline(ctx, pipelineID)
}

// ListArchivedSessionDetails returns composed ContentSessionDetail records for
// archived sessions belonging to a pipeline, sorted newest first.
func (s *ContentSessionService) ListArchivedSessionDetails(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListArchivedByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})
	details := make([]*upal.ContentSessionDetail, 0, len(sessions))
	for i, sess := range sessions {
		sources, err := s.fetches.ListBySession(ctx, sess.ID)
		if err != nil {
			return nil, fmt.Errorf("list sources for session %s: %w", sess.ID, err)
		}
		analysis, _ := s.analyses.GetBySession(ctx, sess.ID)
		wfResults := s.GetWorkflowResults(ctx, sess.ID)
		details = append(details, &upal.ContentSessionDetail{
			ID:              sess.ID,
			PipelineID:      sess.PipelineID,
			SessionNumber:   i + 1,
			Status:          sess.Status,
			TriggerType:     sess.TriggerType,
			SourceCount:     sess.SourceCount,
			Sources:         sources,
			Analysis:        analysis,
			WorkflowResults: wfResults,
			CreatedAt:       sess.CreatedAt,
			ReviewedAt:      sess.ReviewedAt,
			ArchivedAt:      sess.ArchivedAt,
		})
	}
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})
	return details, nil
}

// --- WorkflowResults (in-memory) ---

// SetWorkflowResults stores workflow results for a session, replacing any existing results.
func (s *ContentSessionService) SetWorkflowResults(_ context.Context, sessionID string, results []upal.WorkflowResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflowResults[sessionID] = results
}

// GetWorkflowResults retrieves workflow results for a session.
// Returns an empty slice (not nil) if no results are stored.
func (s *ContentSessionService) GetWorkflowResults(_ context.Context, sessionID string) []upal.WorkflowResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.workflowResults[sessionID]; ok {
		return r
	}
	return []upal.WorkflowResult{}
}

// --- Session Detail (composed views) ---

// GetSessionDetail composes a full ContentSessionDetail from related data.
func (s *ContentSessionService) GetSessionDetail(ctx context.Context, id string) (*upal.ContentSessionDetail, error) {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	sources, err := s.fetches.ListBySession(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list sources for session %s: %w", id, err)
	}

	analysis, _ := s.analyses.GetBySession(ctx, id) // nil if not found

	wfResults := s.GetWorkflowResults(ctx, id)

	// Compute session number: 1-based position among pipeline sessions sorted by created_at.
	sessionNumber := 0
	if sess.PipelineID != "" {
		pipelineSessions, err := s.sessions.ListByPipeline(ctx, sess.PipelineID)
		if err == nil {
			sort.Slice(pipelineSessions, func(i, j int) bool {
				return pipelineSessions[i].CreatedAt.Before(pipelineSessions[j].CreatedAt)
			})
			for i, ps := range pipelineSessions {
				if ps.ID == id {
					sessionNumber = i + 1
					break
				}
			}
		}
	}

	return &upal.ContentSessionDetail{
		ID:              sess.ID,
		PipelineID:      sess.PipelineID,
		SessionNumber:   sessionNumber,
		Status:          sess.Status,
		TriggerType:     sess.TriggerType,
		SourceCount:     sess.SourceCount,
		Sources:         sources,
		Analysis:        analysis,
		WorkflowResults: wfResults,
		CreatedAt:       sess.CreatedAt,
		ReviewedAt:      sess.ReviewedAt,
		ArchivedAt:      sess.ArchivedAt,
	}, nil
}

// ListSessionDetails returns composed ContentSessionDetail records for all
// sessions belonging to a pipeline, sorted by created_at descending (newest first).
func (s *ContentSessionService) ListSessionDetails(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}

	// Sort ascending by created_at first to assign session numbers.
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})

	details := make([]*upal.ContentSessionDetail, 0, len(sessions))
	for i, sess := range sessions {
		sources, err := s.fetches.ListBySession(ctx, sess.ID)
		if err != nil {
			return nil, fmt.Errorf("list sources for session %s: %w", sess.ID, err)
		}

		analysis, _ := s.analyses.GetBySession(ctx, sess.ID) // nil if not found

		wfResults := s.GetWorkflowResults(ctx, sess.ID)

		details = append(details, &upal.ContentSessionDetail{
			ID:              sess.ID,
			PipelineID:      sess.PipelineID,
			SessionNumber:   i + 1, // 1-based
			Status:          sess.Status,
			TriggerType:     sess.TriggerType,
			SourceCount:     sess.SourceCount,
			Sources:         sources,
			Analysis:        analysis,
			WorkflowResults: wfResults,
			CreatedAt:       sess.CreatedAt,
			ReviewedAt:      sess.ReviewedAt,
			ArchivedAt:      sess.ArchivedAt,
		})
	}

	// Reverse to descending (newest first) for the API response.
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})

	return details, nil
}
