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

	// Cache pipeline names to avoid repeated lookups.
	pipelineNames := make(map[string]string)
	lookupPipelineName := func(pipelineID string) string {
		if pipelineID == "" || s.pipelineRepo == nil {
			return ""
		}
		if name, ok := pipelineNames[pipelineID]; ok {
			return name
		}
		if p, err := s.pipelineRepo.Get(ctx, pipelineID); err == nil {
			pipelineNames[pipelineID] = p.Name
			return p.Name
		}
		return ""
	}

	details := make([]*upal.ContentSessionDetail, 0, len(sessions))
	for _, sess := range sessions {
		analysis, _ := s.analyses.GetBySession(ctx, sess.ID)
		wfResults := s.GetWorkflowResults(ctx, sess.ID)
		details = append(details, &upal.ContentSessionDetail{
			ID:               sess.ID,
			PipelineID:       sess.PipelineID,
			Name:             sess.Name,
			PipelineName:     lookupPipelineName(sess.PipelineID),
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
		})
	}
	return details, nil
}

// ListSessionDetailsByStatusIncludeArchived returns composed details including archived sessions.
func (s *ContentSessionService) ListSessionDetailsByStatusIncludeArchived(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListAllByStatus(ctx, status)
	if err != nil {
		return nil, err
	}

	pipelineNames := make(map[string]string)
	lookupPipelineName := func(pipelineID string) string {
		if pipelineID == "" || s.pipelineRepo == nil {
			return ""
		}
		if name, ok := pipelineNames[pipelineID]; ok {
			return name
		}
		if p, err := s.pipelineRepo.Get(ctx, pipelineID); err == nil {
			pipelineNames[pipelineID] = p.Name
			return p.Name
		}
		return ""
	}

	details := make([]*upal.ContentSessionDetail, 0, len(sessions))
	for _, sess := range sessions {
		analysis, _ := s.analyses.GetBySession(ctx, sess.ID)
		wfResults := s.GetWorkflowResults(ctx, sess.ID)
		details = append(details, &upal.ContentSessionDetail{
			ID:               sess.ID,
			PipelineID:       sess.PipelineID,
			Name:             sess.Name,
			PipelineName:     lookupPipelineName(sess.PipelineID),
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
		})
	}
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

	var pipelineName string
	if pipelineID != "" && s.pipelineRepo != nil {
		if p, err := s.pipelineRepo.Get(ctx, pipelineID); err == nil {
			pipelineName = p.Name
		}
	}

	// Build session number lookup from ALL sessions (active + archived).
	allSessions := s.allPipelineSessions(ctx, pipelineID)
	numberOf := make(map[string]int, len(allSessions))
	for i, ps := range allSessions {
		numberOf[ps.ID] = i + 1
	}

	details := make([]*upal.ContentSessionDetail, 0, len(archivedSessions))
	for _, sess := range archivedSessions {
		sources, err := s.fetches.ListBySession(ctx, sess.ID)
		if err != nil {
			return nil, fmt.Errorf("list sources for session %s: %w", sess.ID, err)
		}
		analysis, _ := s.analyses.GetBySession(ctx, sess.ID)
		wfResults := s.GetWorkflowResults(ctx, sess.ID)
		details = append(details, &upal.ContentSessionDetail{
			ID:               sess.ID,
			PipelineID:       sess.PipelineID,
			Name:             sess.Name,
			PipelineName:     pipelineName,
			SessionNumber:    numberOf[sess.ID],
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
			Sources:          sources,
			Analysis:         analysis,
			WorkflowResults:  wfResults,
			CreatedAt:        sess.CreatedAt,
			ReviewedAt:       sess.ReviewedAt,
			ArchivedAt:       sess.ArchivedAt,
		})
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

	pipelineNames := make(map[string]string)
	lookupPipelineName := func(pipelineID string) string {
		if pipelineID == "" || s.pipelineRepo == nil {
			return ""
		}
		if name, ok := pipelineNames[pipelineID]; ok {
			return name
		}
		if p, err := s.pipelineRepo.Get(ctx, pipelineID); err == nil {
			pipelineNames[pipelineID] = p.Name
			return p.Name
		}
		return ""
	}

	details := make([]*upal.ContentSessionDetail, 0, len(archivedSessions))
	for _, sess := range archivedSessions {
		analysis, _ := s.analyses.GetBySession(ctx, sess.ID)
		wfResults := s.GetWorkflowResults(ctx, sess.ID)
		details = append(details, &upal.ContentSessionDetail{
			ID:               sess.ID,
			PipelineID:       sess.PipelineID,
			Name:             sess.Name,
			PipelineName:     lookupPipelineName(sess.PipelineID),
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
		})
	}
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})
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

	sources, err := s.fetches.ListBySession(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list sources for session %s: %w", id, err)
	}

	analysis, _ := s.analyses.GetBySession(ctx, id) // nil if not found

	wfResults := s.GetWorkflowResults(ctx, id)

	// Compute session number: 1-based position among ALL pipeline sessions
	// (active + archived) sorted by created_at, so archived sessions keep
	// their original number.
	sessionNumber := 0
	if sess.PipelineID != "" {
		allSessions := s.allPipelineSessions(ctx, sess.PipelineID)
		for i, ps := range allSessions {
			if ps.ID == id {
				sessionNumber = i + 1
				break
			}
		}
	}

	detail := &upal.ContentSessionDetail{
		ID:               sess.ID,
		PipelineID:       sess.PipelineID,
		Name:             sess.Name,
		SessionNumber:    sessionNumber,
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
		Sources:          sources,
		Analysis:         analysis,
		WorkflowResults:  wfResults,
		CreatedAt:        sess.CreatedAt,
		ReviewedAt:       sess.ReviewedAt,
		ArchivedAt:       sess.ArchivedAt,
	}
	if sess.PipelineID != "" && s.pipelineRepo != nil {
		if p, err := s.pipelineRepo.Get(ctx, sess.PipelineID); err == nil {
			detail.PipelineName = p.Name
		}
	}
	return detail, nil
}

// ListSessionDetails returns composed ContentSessionDetail records for all
// sessions belonging to a pipeline, sorted by created_at descending (newest first).
func (s *ContentSessionService) ListSessionDetails(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}

	var pipelineName string
	if pipelineID != "" && s.pipelineRepo != nil {
		if p, err := s.pipelineRepo.Get(ctx, pipelineID); err == nil {
			pipelineName = p.Name
		}
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
			ID:               sess.ID,
			PipelineID:       sess.PipelineID,
			Name:             sess.Name,
			PipelineName:     pipelineName,
			SessionNumber:    i + 1, // 1-based
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
			Sources:          sources,
			Analysis:         analysis,
			WorkflowResults:  wfResults,
			CreatedAt:        sess.CreatedAt,
			ReviewedAt:       sess.ReviewedAt,
			ArchivedAt:       sess.ArchivedAt,
		})
	}

	// Reverse to descending (newest first) for the API response.
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

	var pipelineName string
	if pipelineID != "" && s.pipelineRepo != nil {
		if p, err := s.pipelineRepo.Get(ctx, pipelineID); err == nil {
			pipelineName = p.Name
		}
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
			ID:               sess.ID,
			PipelineID:       sess.PipelineID,
			Name:             sess.Name,
			PipelineName:     pipelineName,
			SessionNumber:    i + 1,
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
			Sources:          sources,
			Analysis:         analysis,
			WorkflowResults:  wfResults,
			CreatedAt:        sess.CreatedAt,
			ReviewedAt:       sess.ReviewedAt,
			ArchivedAt:       sess.ArchivedAt,
		})
	}

	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})

	return details, nil
}
