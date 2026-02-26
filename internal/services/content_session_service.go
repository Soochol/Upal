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

// ContentSessionService manages content collection sessions and related records.
type ContentSessionService struct {
	sessions        repository.ContentSessionRepository
	fetches         repository.SourceFetchRepository
	analyses        repository.LLMAnalysisRepository
	published       repository.PublishedContentRepository
	surges          repository.SurgeEventRepository
	workflowResults repository.WorkflowResultRepository
	pipelineRepo    repository.PipelineRepository
}

// SetPipelineRepository configures the pipeline repo for resolving pipeline names.
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
	// Auto-generate a default name when not provided (e.g. scheduled/surge sessions).
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

// ListSessionDetailsByStatus returns composed ContentSessionDetail records
// for all sessions matching the given status, across all pipelines.
func (s *ContentSessionService) ListSessionDetailsByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	names := s.newPipelineNameCache(ctx)
	return s.buildDetails(ctx, sessions, names), nil
}

// ListSessionDetailsByStatusIncludeArchived returns composed details including archived sessions.
func (s *ContentSessionService) ListSessionDetailsByStatusIncludeArchived(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListAllByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	names := s.newPipelineNameCache(ctx)
	return s.buildDetails(ctx, sessions, names), nil
}

// ListAllInstanceSessionDetails returns composed ContentSessionDetail records
// for all non-template, non-archived sessions across all pipelines, sorted newest first.
// This powers the unified inbox with a single query instead of per-status calls.
func (s *ContentSessionService) ListAllInstanceSessionDetails(ctx context.Context) ([]*upal.ContentSessionDetail, error) {
	allSessions, err := s.sessions.List(ctx)
	if err != nil {
		return nil, err
	}
	// Filter out templates -- session numbers only apply to execution instances.
	instances := make([]*upal.ContentSession, 0, len(allSessions))
	for _, sess := range allSessions {
		if !sess.IsTemplate {
			instances = append(instances, sess)
		}
	}
	names := s.newPipelineNameCache(ctx)
	details := s.buildDetails(ctx, instances, names)
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})
	return details, nil
}

func (s *ContentSessionService) ListSessionsByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return s.sessions.ListByPipelineAndStatus(ctx, pipelineID, status)
}

// UpdateSessionSettings conditionally updates session configuration fields.
// Only non-zero fields are applied so that partial saves don't destroy data.
// Settings can only be changed while the session is in draft or active status.
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

func (s *ContentSessionService) UpdateSourceFetch(ctx context.Context, sf *upal.SourceFetch) error {
	return s.fetches.Update(ctx, sf)
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

// UpdateAnalysisAngles updates the suggested angles of an existing analysis.
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

// UpdateAngleWorkflow updates the workflow_name and match_type of a single angle by ID.
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

	// Clean up published_content (no FK cascade)
	if err := s.published.DeleteBySession(ctx, id); err != nil {
		return fmt.Errorf("delete published content: %w", err)
	}

	// Delete session (source_fetches + llm_analyses cascade in DB)
	if err := s.sessions.Delete(ctx, id); err != nil {
		return err
	}

	// Clean up workflow results (best-effort — session already deleted)
	if err := s.workflowResults.DeleteBySession(ctx, id); err != nil {
		slog.Warn("failed to clean up workflow results", "session_id", id, "err", err)
	}

	return nil
}

// DeleteSessionsByPipeline removes all sessions (active, archived, templates)
// belonging to a pipeline and their associated data. Used during pipeline deletion.
func (s *ContentSessionService) DeleteSessionsByPipeline(ctx context.Context, pipelineID string) error {
	// Collect all session IDs: active + archived + templates
	active, _ := s.sessions.ListByPipeline(ctx, pipelineID)
	archived, _ := s.sessions.ListArchivedByPipeline(ctx, pipelineID)
	templates, _ := s.sessions.ListTemplatesByPipeline(ctx, pipelineID)

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

// ListArchivedSessionDetails returns composed ContentSessionDetail records for
// archived sessions belonging to a pipeline, sorted newest first.
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
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})
	return details, nil
}

// ListAllArchivedSessionDetails returns composed ContentSessionDetail records
// for ALL archived sessions across all pipelines, sorted newest first.
func (s *ContentSessionService) ListAllArchivedSessionDetails(ctx context.Context) ([]*upal.ContentSessionDetail, error) {
	archivedSessions, err := s.sessions.ListArchived(ctx)
	if err != nil {
		return nil, err
	}
	names := s.newPipelineNameCache(ctx)
	details := s.buildDetails(ctx, archivedSessions, names)
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})
	return details, nil
}

// --- Detail-building helpers ---
//
// These extract the repeated pattern of mapping ContentSession → ContentSessionDetail.
// Two variants exist:
//   - buildDetails: lightweight (no source fetches, no session numbers)
//   - buildFullDetails: includes source fetches and session numbers (may return error)

// pipelineNameCache caches pipeline ID → name lookups within a single request.
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

// sessionNumberMap returns a map of session ID → 1-based position among all
// pipeline sessions (active + archived), so archived sessions keep their
// original number. Returns nil for empty pipelineID (cross-pipeline queries).
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

// sessionToDetail maps a ContentSession to a ContentSessionDetail with analysis
// and workflow results, but without source fetches or session numbers.
func (s *ContentSessionService) sessionToDetail(
	ctx context.Context, sess *upal.ContentSession, names *pipelineNameCache,
) *upal.ContentSessionDetail {
	analysis, _ := s.analyses.GetBySession(ctx, sess.ID)
	wfResults := s.GetWorkflowResults(ctx, sess.ID)
	return &upal.ContentSessionDetail{
		ID:               sess.ID,
		PipelineID:       sess.PipelineID,
		Name:             sess.Name,
		PipelineName:     names.lookup(sess.PipelineID),
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

// buildDetails maps a slice of sessions to lightweight details (no source fetches, no session numbers).
func (s *ContentSessionService) buildDetails(
	ctx context.Context, sessions []*upal.ContentSession, names *pipelineNameCache,
) []*upal.ContentSessionDetail {
	details := make([]*upal.ContentSessionDetail, 0, len(sessions))
	for _, sess := range sessions {
		details = append(details, s.sessionToDetail(ctx, sess, names))
	}
	return details
}

// buildFullDetails maps sessions to full details including source fetches and session numbers.
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

// allPipelineSessions returns active + archived instance sessions for a pipeline,
// sorted by created_at ascending, so session numbers are stable.
// Templates are excluded (ListByPipeline filters !IsTemplate) because session
// numbers only apply to execution instances, not reusable templates.
func (s *ContentSessionService) allPipelineSessions(ctx context.Context, pipelineID string) []*upal.ContentSession {
	active, _ := s.sessions.ListByPipeline(ctx, pipelineID)
	archived, _ := s.sessions.ListArchivedByPipeline(ctx, pipelineID)
	all := append(active, archived...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})
	return all
}

// --- WorkflowResults ---

// SetWorkflowResults stores workflow results for a session, replacing any existing results.
func (s *ContentSessionService) SetWorkflowResults(ctx context.Context, sessionID string, results []upal.WorkflowResult) {
	_ = s.workflowResults.Save(ctx, sessionID, results)
}

// GetWorkflowResults retrieves workflow results for a session.
// Returns an empty slice (not nil) if no results are stored.
func (s *ContentSessionService) GetWorkflowResults(ctx context.Context, sessionID string) []upal.WorkflowResult {
	results, _ := s.workflowResults.GetBySession(ctx, sessionID)
	return results
}

// --- Session Detail (composed views) ---

// GetSessionDetail composes a full ContentSessionDetail from related data.
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

// ListSessionDetails returns composed ContentSessionDetail records for all
// sessions belonging to a pipeline, sorted by created_at descending (newest first).
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
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})
	return details, nil
}

// ListSessionDetailsByPipelineAndStatus returns composed session details for a
// pipeline, optionally filtered by status. If status is empty, all sessions
// are returned (equivalent to ListSessionDetails).
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

// --- Template Sessions ---

// ListTemplatesByPipeline returns template sessions belonging to a pipeline.
func (s *ContentSessionService) ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return s.sessions.ListTemplatesByPipeline(ctx, pipelineID)
}

// ListTemplateDetailsByPipeline returns composed ContentSessionDetail records
// for template sessions belonging to a pipeline, sorted newest first.
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
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})
	return details, nil
}
