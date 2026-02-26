package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
)

var _ ports.ContentSessionPort = (*ContentSessionService)(nil)

type ContentSessionService struct {
	sessions        repository.ContentSessionRepository
	fetches         repository.SourceFetchRepository
	analyses        repository.LLMAnalysisRepository
	published       repository.PublishedContentRepository
	surges          repository.SurgeEventRepository
	workflowResults repository.WorkflowResultRepository
	pipelineRepo    repository.PipelineRepository
}

func (s *ContentSessionService) SetPipelineRepository(repo repository.PipelineRepository) {
	s.pipelineRepo = repo
}

func NewContentSessionService(
	sessions repository.ContentSessionRepository,
	fetches repository.SourceFetchRepository,
	analyses repository.LLMAnalysisRepository,
	published repository.PublishedContentRepository,
	surges repository.SurgeEventRepository,
	workflowResults repository.WorkflowResultRepository,
) *ContentSessionService {
	return &ContentSessionService{
		sessions:        sessions,
		fetches:         fetches,
		analyses:        analyses,
		published:       published,
		surges:          surges,
		workflowResults: workflowResults,
	}
}

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
	if sess.Name == "" {
		allSessions := s.allPipelineSessions(ctx, sess.PipelineID)
		sess.Name = fmt.Sprintf("Session %d", len(allSessions)+1)
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

func (s *ContentSessionService) ListSessionDetailsByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	names := s.newPipelineNameCache(ctx)
	return s.buildDetails(ctx, sessions, names), nil
}

func (s *ContentSessionService) ListSessionDetailsByStatusIncludeArchived(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListAllByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	names := s.newPipelineNameCache(ctx)
	return s.buildDetails(ctx, sessions, names), nil
}

func (s *ContentSessionService) ListAllInstanceSessionDetails(ctx context.Context) ([]*upal.ContentSessionDetail, error) {
	allSessions, err := s.sessions.List(ctx)
	if err != nil {
		return nil, err
	}
	instances := make([]*upal.ContentSession, 0, len(allSessions))
	for _, sess := range allSessions {
		if !sess.IsTemplate {
			instances = append(instances, sess)
		}
	}
	names := s.newPipelineNameCache(ctx)
	details := s.buildDetails(ctx, instances, names)
	sortDetailsNewestFirst(details)
	return details, nil
}

func (s *ContentSessionService) ListSessionsByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return s.sessions.ListByPipelineAndStatus(ctx, pipelineID, status)
}

func (s *ContentSessionService) UpdateSessionSettings(ctx context.Context, id string, settings upal.SessionSettings) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	if sess.Status != upal.SessionDraft && sess.Status != upal.SessionActive {
		return fmt.Errorf("session %q: settings can only be changed in draft or active status", id)
	}
	if settings.Name != "" {
		sess.Name = settings.Name
	}
	if settings.Sources != nil {
		sess.Sources = settings.Sources
	}
	if settings.Schedule != "" {
		sess.Schedule = settings.Schedule
	}
	if settings.ClearSchedule {
		sess.Schedule = ""
	}
	if settings.Model != "" {
		sess.Model = settings.Model
	}
	if settings.Workflows != nil {
		sess.Workflows = settings.Workflows
	}
	if settings.Context != nil {
		sess.Context = settings.Context
	}
	return s.sessions.Update(ctx, sess)
}

func (s *ContentSessionService) UpdateSession(ctx context.Context, sess *upal.ContentSession) error {
	return s.sessions.Update(ctx, sess)
}

func (s *ContentSessionService) UpdateSessionStatus(ctx context.Context, id string, status upal.ContentSessionStatus) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	sess.Status = status
	return s.sessions.Update(ctx, sess)
}

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

func (s *ContentSessionService) RecordSourceFetch(ctx context.Context, sf *upal.SourceFetch) error {
	if sf.ID == "" {
		sf.ID = upal.GenerateID("sfetch")
	}
	sf.FetchedAt = time.Now()
	return s.fetches.Create(ctx, sf)
}

func (s *ContentSessionService) UpdateSourceFetch(ctx context.Context, sf *upal.SourceFetch) error {
	return s.fetches.Update(ctx, sf)
}

func (s *ContentSessionService) ListSourceFetches(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	return s.fetches.ListBySession(ctx, sessionID)
}

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

func (s *ContentSessionService) UpdateAnalysisAngles(ctx context.Context, sessionID string, angles []upal.ContentAngle) error {
	analysis, err := s.analyses.GetBySession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get analysis for session %s: %w", sessionID, err)
	}
	if analysis == nil {
		return fmt.Errorf("no analysis found for session %s", sessionID)
	}
	analysis.SuggestedAngles = angles
	return s.analyses.Update(ctx, analysis)
}

func (s *ContentSessionService) UpdateAngleWorkflow(ctx context.Context, sessionID, angleID, workflowName string) error {
	analysis, err := s.analyses.GetBySession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get analysis for session %s: %w", sessionID, err)
	}
	if analysis == nil {
		return fmt.Errorf("no analysis found for session %s", sessionID)
	}
	found := false
	for i := range analysis.SuggestedAngles {
		if analysis.SuggestedAngles[i].ID == angleID {
			analysis.SuggestedAngles[i].WorkflowName = workflowName
			analysis.SuggestedAngles[i].MatchType = "manual"
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("angle %s not found in session %s", angleID, sessionID)
	}
	return s.analyses.Update(ctx, analysis)
}

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

func (s *ContentSessionService) ArchiveSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	if sess.ArchivedAt != nil {
		return fmt.Errorf("session %q: %w", id, upal.ErrAlreadyArchived)
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
		return fmt.Errorf("session %q: %w", id, upal.ErrNotArchived)
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
		return fmt.Errorf("session %q: %w", id, upal.ErrMustBeArchived)
	}

	if err := s.published.DeleteBySession(ctx, id); err != nil {
		return fmt.Errorf("delete published content: %w", err)
	}
	if err := s.sessions.Delete(ctx, id); err != nil {
		return err
	}
	if err := s.workflowResults.DeleteBySession(ctx, id); err != nil {
		slog.Warn("failed to clean up workflow results", "session_id", id, "err", err)
	}

	return nil
}

func (s *ContentSessionService) DeleteSessionsByPipeline(ctx context.Context, pipelineID string) error {
	active, err := s.sessions.ListByPipeline(ctx, pipelineID)
	if err != nil {
		return fmt.Errorf("list active sessions for pipeline %s: %w", pipelineID, err)
	}
	archived, err := s.sessions.ListArchivedByPipeline(ctx, pipelineID)
	if err != nil {
		return fmt.Errorf("list archived sessions for pipeline %s: %w", pipelineID, err)
	}
	templates, err := s.sessions.ListTemplatesByPipeline(ctx, pipelineID)
	if err != nil {
		return fmt.Errorf("list template sessions for pipeline %s: %w", pipelineID, err)
	}

	seen := make(map[string]bool)
	var ids []string
	for _, list := range [][]*upal.ContentSession{active, archived, templates} {
		for _, sess := range list {
			if !seen[sess.ID] {
				seen[sess.ID] = true
				ids = append(ids, sess.ID)
			}
		}
	}

	for _, id := range ids {
		_ = s.published.DeleteBySession(ctx, id)
		_ = s.workflowResults.DeleteBySession(ctx, id)
		if err := s.sessions.Delete(ctx, id); err != nil {
			slog.Warn("failed to delete session during pipeline cleanup", "session_id", id, "err", err)
		}
	}
	return nil
}

func (s *ContentSessionService) ListArchivedByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return s.sessions.ListArchivedByPipeline(ctx, pipelineID)
}

func (s *ContentSessionService) ListArchivedSessionDetails(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error) {
	archivedSessions, err := s.sessions.ListArchivedByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	names := s.newPipelineNameCache(ctx)
	numbers := s.sessionNumberMap(ctx, pipelineID)
	details, err := s.buildFullDetails(ctx, archivedSessions, names, numbers)
	if err != nil {
		return nil, err
	}
	sortDetailsNewestFirst(details)
	return details, nil
}

func (s *ContentSessionService) ListAllArchivedSessionDetails(ctx context.Context) ([]*upal.ContentSessionDetail, error) {
	archivedSessions, err := s.sessions.ListArchived(ctx)
	if err != nil {
		return nil, err
	}
	names := s.newPipelineNameCache(ctx)
	details := s.buildDetails(ctx, archivedSessions, names)
	sortDetailsNewestFirst(details)
	return details, nil
}

func sortDetailsNewestFirst(details []*upal.ContentSessionDetail) {
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})
}

type pipelineNameCache struct {
	svc   *ContentSessionService
	ctx   context.Context
	cache map[string]string
}

func (s *ContentSessionService) newPipelineNameCache(ctx context.Context) *pipelineNameCache {
	return &pipelineNameCache{svc: s, ctx: ctx, cache: make(map[string]string)}
}

func (c *pipelineNameCache) lookup(pipelineID string) string {
	if pipelineID == "" || c.svc.pipelineRepo == nil {
		return ""
	}
	if name, ok := c.cache[pipelineID]; ok {
		return name
	}
	if p, err := c.svc.pipelineRepo.Get(c.ctx, pipelineID); err == nil {
		c.cache[pipelineID] = p.Name
		return p.Name
	}
	return ""
}

func (s *ContentSessionService) sessionNumberMap(ctx context.Context, pipelineID string) map[string]int {
	if pipelineID == "" {
		return nil
	}
	all := s.allPipelineSessions(ctx, pipelineID)
	numbers := make(map[string]int, len(all))
	for i, ps := range all {
		numbers[ps.ID] = i + 1
	}
	return numbers
}

func (s *ContentSessionService) sessionToDetail(
	ctx context.Context, sess *upal.ContentSession, names *pipelineNameCache,
) *upal.ContentSessionDetail {
	analysis, _ := s.analyses.GetBySession(ctx, sess.ID)
	wfResults := s.GetWorkflowResults(ctx, sess.ID)

	var sessionName string
	if sess.ParentSessionID != "" {
		if parent, err := s.sessions.Get(ctx, sess.ParentSessionID); err == nil {
			sessionName = parent.Name
		}
	}

	return &upal.ContentSessionDetail{
		ID:               sess.ID,
		PipelineID:       sess.PipelineID,
		Name:             sess.Name,
		PipelineName:     names.lookup(sess.PipelineID),
		SessionName:      sessionName,
		Status:           sess.Status,
		TriggerType:      sess.TriggerType,
		SourceCount:      sess.SourceCount,
		IsTemplate:       sess.IsTemplate,
		ParentSessionID:  sess.ParentSessionID,
		ScheduleID:       sess.ScheduleID,
		SessionSources:   sess.Sources,
		Schedule:         sess.Schedule,
		Model:            sess.Model,
		SessionWorkflows: sess.Workflows,
		SessionContext:   sess.Context,
		Analysis:         analysis,
		WorkflowResults:  wfResults,
		CreatedAt:        sess.CreatedAt,
		ReviewedAt:       sess.ReviewedAt,
		ArchivedAt:       sess.ArchivedAt,
	}
}

func (s *ContentSessionService) buildDetails(
	ctx context.Context, sessions []*upal.ContentSession, names *pipelineNameCache,
) []*upal.ContentSessionDetail {
	details := make([]*upal.ContentSessionDetail, 0, len(sessions))
	for _, sess := range sessions {
		details = append(details, s.sessionToDetail(ctx, sess, names))
	}
	return details
}

func (s *ContentSessionService) buildFullDetails(
	ctx context.Context, sessions []*upal.ContentSession,
	names *pipelineNameCache, numbers map[string]int,
) ([]*upal.ContentSessionDetail, error) {
	details := make([]*upal.ContentSessionDetail, 0, len(sessions))
	for _, sess := range sessions {
		d := s.sessionToDetail(ctx, sess, names)
		sources, err := s.fetches.ListBySession(ctx, sess.ID)
		if err != nil {
			return nil, fmt.Errorf("list sources for session %s: %w", sess.ID, err)
		}
		d.Sources = sources
		if numbers != nil {
			d.SessionNumber = numbers[sess.ID]
		}
		details = append(details, d)
	}
	return details, nil
}

func (s *ContentSessionService) allPipelineSessions(ctx context.Context, pipelineID string) []*upal.ContentSession {
	active, _ := s.sessions.ListByPipeline(ctx, pipelineID)
	archived, _ := s.sessions.ListArchivedByPipeline(ctx, pipelineID)
	all := append(active, archived...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})
	return all
}

func (s *ContentSessionService) SetWorkflowResults(ctx context.Context, sessionID string, results []upal.WorkflowResult) {
	_ = s.workflowResults.Save(ctx, sessionID, results)
}

func (s *ContentSessionService) GetWorkflowResults(ctx context.Context, sessionID string) []upal.WorkflowResult {
	results, _ := s.workflowResults.GetBySession(ctx, sessionID)
	return results
}

func (s *ContentSessionService) GetSessionDetail(ctx context.Context, id string) (*upal.ContentSessionDetail, error) {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	names := s.newPipelineNameCache(ctx)
	numbers := s.sessionNumberMap(ctx, sess.PipelineID)
	details, err := s.buildFullDetails(ctx, []*upal.ContentSession{sess}, names, numbers)
	if err != nil {
		return nil, err
	}
	return details[0], nil
}

func (s *ContentSessionService) ListSessionDetails(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	names := s.newPipelineNameCache(ctx)
	numbers := s.sessionNumberMap(ctx, pipelineID)
	details, err := s.buildFullDetails(ctx, sessions, names, numbers)
	if err != nil {
		return nil, err
	}
	sortDetailsNewestFirst(details)
	return details, nil
}

func (s *ContentSessionService) ListSessionDetailsByPipelineAndStatus(
	ctx context.Context, pipelineID string, status upal.ContentSessionStatus,
) ([]*upal.ContentSessionDetail, error) {
	details, err := s.ListSessionDetails(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	if status == "" {
		return details, nil
	}
	filtered := make([]*upal.ContentSessionDetail, 0, len(details))
	for _, d := range details {
		if d.Status == status {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}

func (s *ContentSessionService) ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return s.sessions.ListTemplatesByPipeline(ctx, pipelineID)
}

func (s *ContentSessionService) ListTemplateDetailsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListTemplatesByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	names := s.newPipelineNameCache(ctx)
	numbers := s.sessionNumberMap(ctx, pipelineID)
	details, err := s.buildFullDetails(ctx, sessions, names, numbers)
	if err != nil {
		return nil, err
	}
	sortDetailsNewestFirst(details)
	return details, nil
}
