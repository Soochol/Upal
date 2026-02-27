package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// RunService manages the lifecycle of Run entities and their children
// (source fetches, analyses, workflow runs, published content, surges).
type RunService struct {
	runs      repository.SessionRunRepository
	sessions  repository.SessionRepository
	fetches   repository.SourceFetchRepository
	analyses  repository.LLMAnalysisRepository
	published repository.PublishedContentRepository
	surges    repository.SurgeEventRepository
	wfRuns    repository.WorkflowRunRepository
}

// NewRunService creates a new RunService.
func NewRunService(
	runs repository.SessionRunRepository,
	sessions repository.SessionRepository,
	fetches repository.SourceFetchRepository,
	analyses repository.LLMAnalysisRepository,
	published repository.PublishedContentRepository,
	surges repository.SurgeEventRepository,
	wfRuns repository.WorkflowRunRepository,
) *RunService {
	return &RunService{
		runs:      runs,
		sessions:  sessions,
		fetches:   fetches,
		analyses:  analyses,
		published: published,
		surges:    surges,
		wfRuns:    wfRuns,
	}
}

// ---------------------------------------------------------------------------
// Run CRUD
// ---------------------------------------------------------------------------

// CreateRun creates a new Run for the given session.
func (s *RunService) CreateRun(ctx context.Context, sessionID, triggerType string) (*upal.Run, error) {
	if _, err := s.sessions.Get(ctx, sessionID); err != nil {
		return nil, fmt.Errorf("session %q: %w", sessionID, err)
	}
	if triggerType == "" {
		triggerType = "manual"
	}
	run := &upal.Run{
		ID:          upal.GenerateID("run"),
		SessionID:   sessionID,
		Status:      upal.SessionRunCollecting,
		TriggerType: triggerType,
		CreatedAt:   time.Now(),
	}
	if err := s.runs.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// CreateRunWithConfig creates a new Run with full configuration.
func (s *RunService) CreateRunWithConfig(ctx context.Context, sessionID, triggerType, name string,
	sources []upal.SessionSource, workflows []upal.SessionWorkflow, sessionCtx *upal.SessionContext, schedule string,
) (*upal.Run, error) {
	if _, err := s.sessions.Get(ctx, sessionID); err != nil {
		return nil, fmt.Errorf("session %q: %w", sessionID, err)
	}
	if triggerType == "" {
		triggerType = "manual"
	}
	run := &upal.Run{
		ID:          upal.GenerateID("run"),
		SessionID:   sessionID,
		Name:        name,
		Status:      upal.SessionRunCollecting,
		TriggerType: triggerType,
		Sources:     sources,
		Workflows:   workflows,
		Context:     sessionCtx,
		Schedule:    schedule,
		CreatedAt:   time.Now(),
	}
	if err := s.runs.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// UpdateRunConfig updates the configuration of an existing Run.
func (s *RunService) UpdateRunConfig(ctx context.Context, runID, name string,
	sources []upal.SessionSource, workflows []upal.SessionWorkflow, sessionCtx *upal.SessionContext, schedule string,
) error {
	run, err := s.runs.Get(ctx, runID)
	if err != nil {
		return err
	}
	run.Name = name
	run.Sources = sources
	run.Workflows = workflows
	run.Context = sessionCtx
	run.Schedule = schedule
	return s.runs.Update(ctx, run)
}

// ToggleRunSchedule sets the schedule_active flag on a Run.
func (s *RunService) ToggleRunSchedule(ctx context.Context, runID string, active bool) error {
	run, err := s.runs.Get(ctx, runID)
	if err != nil {
		return err
	}
	run.ScheduleActive = active
	return s.runs.Update(ctx, run)
}

// GetRun retrieves a Run by ID.
func (s *RunService) GetRun(ctx context.Context, id string) (*upal.Run, error) {
	return s.runs.Get(ctx, id)
}

// GetRunDetail retrieves a Run with all related data composed inline.
func (s *RunService) GetRunDetail(ctx context.Context, id string) (*upal.RunDetail, error) {
	run, err := s.runs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	detail := s.runToDetail(ctx, run)
	sources, _ := s.fetches.ListBySession(ctx, run.ID)
	detail.Sources = sources
	return detail, nil
}

// ListRuns returns all runs as RunDetail, sorted newest first.
func (s *RunService) ListRuns(ctx context.Context) ([]*upal.RunDetail, error) {
	runs, err := s.runs.List(ctx)
	if err != nil {
		return nil, err
	}
	cache := s.newSessionNameCache(ctx)
	details := make([]*upal.RunDetail, 0, len(runs))
	for _, r := range runs {
		details = append(details, s.runToDetailCached(ctx, r, cache))
	}
	sortRunDetailsNewestFirst(details)
	return details, nil
}

// ListRunsBySession returns runs for a specific session, sorted newest first.
func (s *RunService) ListRunsBySession(ctx context.Context, sessionID string) ([]*upal.RunDetail, error) {
	runs, err := s.runs.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	cache := s.newSessionNameCache(ctx)
	details := make([]*upal.RunDetail, 0, len(runs))
	for _, r := range runs {
		details = append(details, s.runToDetailCached(ctx, r, cache))
	}
	sortRunDetailsNewestFirst(details)
	return details, nil
}

// ListRunsByStatus returns runs with the given status, sorted newest first.
func (s *RunService) ListRunsByStatus(ctx context.Context, status upal.SessionRunStatus) ([]*upal.RunDetail, error) {
	runs, err := s.runs.ListByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	cache := s.newSessionNameCache(ctx)
	details := make([]*upal.RunDetail, 0, len(runs))
	for _, r := range runs {
		details = append(details, s.runToDetailCached(ctx, r, cache))
	}
	sortRunDetailsNewestFirst(details)
	return details, nil
}

// UpdateRunStatus changes the status of an existing Run.
func (s *RunService) UpdateRunStatus(ctx context.Context, id string, status upal.SessionRunStatus) error {
	run, err := s.runs.Get(ctx, id)
	if err != nil {
		return err
	}
	run.Status = status
	return s.runs.Update(ctx, run)
}

// UpdateRunSourceCount sets the source count for a Run.
func (s *RunService) UpdateRunSourceCount(ctx context.Context, id string, count int) error {
	run, err := s.runs.Get(ctx, id)
	if err != nil {
		return err
	}
	run.SourceCount = count
	return s.runs.Update(ctx, run)
}

// ApproveRun marks a Run as approved.
func (s *RunService) ApproveRun(ctx context.Context, id string) error {
	run, err := s.runs.Get(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now()
	run.Status = upal.SessionRunApproved
	run.ReviewedAt = &now
	return s.runs.Update(ctx, run)
}

// RejectRun marks a Run as rejected.
func (s *RunService) RejectRun(ctx context.Context, id string) error {
	run, err := s.runs.Get(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now()
	run.Status = upal.SessionRunRejected
	run.ReviewedAt = &now
	return s.runs.Update(ctx, run)
}

// DeleteRun removes a Run and all its children (cascade).
func (s *RunService) DeleteRun(ctx context.Context, id string) error {
	if _, err := s.runs.Get(ctx, id); err != nil {
		return err
	}
	// Published content failure is hard — must not leave orphaned records.
	if err := s.published.DeleteBySession(ctx, id); err != nil {
		return fmt.Errorf("delete published content: %w", err)
	}
	s.cleanupRunChildren(ctx, id)
	return s.runs.Delete(ctx, id)
}

// DeleteRunsBySession removes all runs for a given session (cascade each).
func (s *RunService) DeleteRunsBySession(ctx context.Context, sessionID string) error {
	runs, err := s.runs.ListBySession(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, r := range runs {
		if err := s.published.DeleteBySession(ctx, r.ID); err != nil {
			slog.Warn("session cleanup: published content", "run_id", r.ID, "err", err)
		}
		s.cleanupRunChildren(ctx, r.ID)
		if err := s.runs.Delete(ctx, r.ID); err != nil {
			slog.Warn("session cleanup: run", "run_id", r.ID, "err", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Source fetch methods
// ---------------------------------------------------------------------------

// RecordSourceFetch persists a new source fetch record.
func (s *RunService) RecordSourceFetch(ctx context.Context, sf *upal.SourceFetch) error {
	if sf.ID == "" {
		sf.ID = upal.GenerateID("sfetch")
	}
	sf.FetchedAt = time.Now()
	return s.fetches.Create(ctx, sf)
}

// UpdateSourceFetch updates an existing source fetch record.
func (s *RunService) UpdateSourceFetch(ctx context.Context, sf *upal.SourceFetch) error {
	return s.fetches.Update(ctx, sf)
}

// ListSourceFetches returns all source fetches for a given run ID.
// Note: uses SessionID field on SourceFetch (will be renamed to RunID later).
func (s *RunService) ListSourceFetches(ctx context.Context, runID string) ([]*upal.SourceFetch, error) {
	return s.fetches.ListBySession(ctx, runID)
}

// ---------------------------------------------------------------------------
// Analysis methods
// ---------------------------------------------------------------------------

// RecordAnalysis persists a new LLM analysis record.
func (s *RunService) RecordAnalysis(ctx context.Context, a *upal.LLMAnalysis) error {
	if a.ID == "" {
		a.ID = upal.GenerateID("anlys")
	}
	a.CreatedAt = time.Now()
	return s.analyses.Create(ctx, a)
}

// GetAnalysis retrieves the LLM analysis for a given run ID.
func (s *RunService) GetAnalysis(ctx context.Context, runID string) (*upal.LLMAnalysis, error) {
	return s.analyses.GetBySession(ctx, runID)
}

// UpdateAnalysis updates the summary and insights of an existing analysis.
func (s *RunService) UpdateAnalysis(ctx context.Context, runID string, summary string, insights []string) error {
	analysis, err := s.analyses.GetBySession(ctx, runID)
	if err != nil {
		return err
	}
	if analysis == nil {
		return fmt.Errorf("no analysis found for run %s", runID)
	}
	analysis.Summary = summary
	analysis.Insights = insights
	return s.analyses.Update(ctx, analysis)
}

// UpdateAnalysisAngles replaces all suggested angles for a run's analysis.
func (s *RunService) UpdateAnalysisAngles(ctx context.Context, runID string, angles []upal.ContentAngle) error {
	analysis, err := s.analyses.GetBySession(ctx, runID)
	if err != nil {
		return fmt.Errorf("get analysis for run %s: %w", runID, err)
	}
	if analysis == nil {
		return fmt.Errorf("no analysis found for run %s", runID)
	}
	analysis.SuggestedAngles = angles
	return s.analyses.Update(ctx, analysis)
}

// UpdateAngleWorkflow sets the workflow for a specific angle in a run's analysis.
func (s *RunService) UpdateAngleWorkflow(ctx context.Context, runID, angleID, workflowName string) error {
	analysis, err := s.analyses.GetBySession(ctx, runID)
	if err != nil {
		return fmt.Errorf("get analysis for run %s: %w", runID, err)
	}
	if analysis == nil {
		return fmt.Errorf("no analysis found for run %s", runID)
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
		return fmt.Errorf("angle %s not found in run %s", angleID, runID)
	}
	return s.analyses.Update(ctx, analysis)
}

// ---------------------------------------------------------------------------
// Workflow run methods
// ---------------------------------------------------------------------------

// SetWorkflowRuns saves workflow execution results for a Run.
func (s *RunService) SetWorkflowRuns(ctx context.Context, runID string, results []upal.WorkflowRun) {
	_ = s.wfRuns.Save(ctx, runID, results)
}

// GetWorkflowRuns retrieves workflow execution results for a Run.
func (s *RunService) GetWorkflowRuns(ctx context.Context, runID string) []upal.WorkflowRun {
	results, _ := s.wfRuns.GetByRun(ctx, runID)
	return results
}

// ---------------------------------------------------------------------------
// Published content methods
// ---------------------------------------------------------------------------

// RecordPublished persists a new published content record.
func (s *RunService) RecordPublished(ctx context.Context, pc *upal.PublishedContent) error {
	if pc.ID == "" {
		pc.ID = upal.GenerateID("pub")
	}
	pc.PublishedAt = time.Now()
	return s.published.Create(ctx, pc)
}

// ListPublished returns all published content.
func (s *RunService) ListPublished(ctx context.Context) ([]*upal.PublishedContent, error) {
	return s.published.List(ctx)
}

// ListPublishedByRun returns published content for a specific run.
// Note: uses SessionID field on PublishedContent (will be renamed later).
func (s *RunService) ListPublishedByRun(ctx context.Context, runID string) ([]*upal.PublishedContent, error) {
	return s.published.ListBySession(ctx, runID)
}

// ListPublishedByChannel returns published content for a specific channel.
func (s *RunService) ListPublishedByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error) {
	return s.published.ListByChannel(ctx, channel)
}

// ---------------------------------------------------------------------------
// Surge event methods
// ---------------------------------------------------------------------------

// CreateSurge persists a new surge event.
func (s *RunService) CreateSurge(ctx context.Context, se *upal.SurgeEvent) error {
	if se.ID == "" {
		se.ID = upal.GenerateID("surge")
	}
	se.CreatedAt = time.Now()
	return s.surges.Create(ctx, se)
}

// ListSurges returns all surge events.
func (s *RunService) ListSurges(ctx context.Context) ([]*upal.SurgeEvent, error) {
	return s.surges.List(ctx)
}

// ListActiveSurges returns undismissed surge events.
func (s *RunService) ListActiveSurges(ctx context.Context) ([]*upal.SurgeEvent, error) {
	return s.surges.ListActive(ctx)
}

// DismissSurge marks a surge event as dismissed.
func (s *RunService) DismissSurge(ctx context.Context, id string) error {
	se, err := s.surges.Get(ctx, id)
	if err != nil {
		return err
	}
	se.Dismissed = true
	return s.surges.Update(ctx, se)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// runToDetail composes a RunDetail from a Run (no session name cache).
func (s *RunService) runToDetail(ctx context.Context, run *upal.Run) *upal.RunDetail {
	detail := &upal.RunDetail{Run: *run}
	if sess, err := s.sessions.Get(ctx, run.SessionID); err == nil {
		detail.SessionName = sess.Name
	}
	analysis, _ := s.analyses.GetBySession(ctx, run.ID)
	detail.Analysis = analysis
	wfRuns, _ := s.wfRuns.GetByRun(ctx, run.ID)
	detail.WorkflowRuns = wfRuns
	return detail
}

// runToDetailCached composes a RunDetail using a session name cache.
func (s *RunService) runToDetailCached(ctx context.Context, run *upal.Run, cache *sessionNameCache) *upal.RunDetail {
	detail := &upal.RunDetail{Run: *run}
	detail.SessionName = cache.lookup(run.SessionID)
	analysis, _ := s.analyses.GetBySession(ctx, run.ID)
	detail.Analysis = analysis
	wfRuns, _ := s.wfRuns.GetByRun(ctx, run.ID)
	detail.WorkflowRuns = wfRuns
	return detail
}

// sessionNameCache avoids repeated session lookups when listing runs.
type sessionNameCache struct {
	svc   *RunService
	ctx   context.Context
	cache map[string]string
}

func (s *RunService) newSessionNameCache(ctx context.Context) *sessionNameCache {
	return &sessionNameCache{svc: s, ctx: ctx, cache: make(map[string]string)}
}

func (c *sessionNameCache) lookup(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	if name, ok := c.cache[sessionID]; ok {
		return name
	}
	if sess, err := c.svc.sessions.Get(c.ctx, sessionID); err == nil {
		c.cache[sessionID] = sess.Name
		return sess.Name
	}
	return ""
}

// cleanupRunChildren removes secondary child records for a run. Errors are
// logged but do not halt the cascade so the parent run can still be deleted.
func (s *RunService) cleanupRunChildren(ctx context.Context, id string) {
	if err := s.wfRuns.DeleteByRun(ctx, id); err != nil {
		slog.Warn("cascade delete: workflow runs", "run_id", id, "err", err)
	}
	if err := s.fetches.DeleteBySession(ctx, id); err != nil {
		slog.Warn("cascade delete: source fetches", "run_id", id, "err", err)
	}
	if err := s.analyses.DeleteBySession(ctx, id); err != nil {
		slog.Warn("cascade delete: llm analyses", "run_id", id, "err", err)
	}
}

func sortRunDetailsNewestFirst(details []*upal.RunDetail) {
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})
}
