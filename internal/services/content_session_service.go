package services

import (
	"context"
	"fmt"
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
}

func NewContentSessionService(
	sessions repository.ContentSessionRepository,
	fetches repository.SourceFetchRepository,
	analyses repository.LLMAnalysisRepository,
	published repository.PublishedContentRepository,
	surges repository.SurgeEventRepository,
) *ContentSessionService {
	return &ContentSessionService{
		sessions:  sessions,
		fetches:   fetches,
		analyses:  analyses,
		published: published,
		surges:    surges,
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
