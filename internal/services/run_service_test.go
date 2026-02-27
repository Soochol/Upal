package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRunEnv bundles all repositories + services needed for run tests.
type testRunEnv struct {
	sessionRepo   *repository.MemorySessionRepository
	runRepo       *repository.MemorySessionRunRepository
	fetchRepo     *repository.MemorySourceFetchRepository
	analysisRepo  *repository.MemoryLLMAnalysisRepository
	publishedRepo *repository.MemoryPublishedContentRepository
	surgeRepo     *repository.MemorySurgeEventRepository
	wfRunRepo     *repository.MemoryWorkflowRunRepository
	sessSvc       *services.SessionService
	runSvc        *services.RunService
}

func newTestRunEnv() *testRunEnv {
	sessRepo := repository.NewMemorySessionRepository()
	runRepo := repository.NewMemorySessionRunRepository()
	fetchRepo := repository.NewMemorySourceFetchRepository()
	analysisRepo := repository.NewMemoryLLMAnalysisRepository()
	publishedRepo := repository.NewMemoryPublishedContentRepository()
	surgeRepo := repository.NewMemorySurgeEventRepository()
	wfRunRepo := repository.NewMemoryWorkflowRunRepository()

	return &testRunEnv{
		sessionRepo:   sessRepo,
		runRepo:       runRepo,
		fetchRepo:     fetchRepo,
		analysisRepo:  analysisRepo,
		publishedRepo: publishedRepo,
		surgeRepo:     surgeRepo,
		wfRunRepo:     wfRunRepo,
		sessSvc:       services.NewSessionService(sessRepo),
		runSvc: services.NewRunService(
			runRepo, sessRepo, fetchRepo, analysisRepo,
			publishedRepo, surgeRepo, wfRunRepo,
		),
	}
}

// createTestSession is a helper that creates a session and returns it.
func (e *testRunEnv) createTestSession(t *testing.T, name string) *upal.Session {
	t.Helper()
	sess, err := e.sessSvc.Create(context.Background(), &upal.Session{Name: name})
	require.NoError(t, err)
	return sess
}

func TestRunService_CreateRun(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test Session")
	run, err := env.runSvc.CreateRun(ctx, sess.ID, "manual")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(run.ID, "run-"))
	assert.Equal(t, upal.SessionRunDraft, run.Status)
	assert.Equal(t, sess.ID, run.SessionID)
	assert.Equal(t, "manual", run.TriggerType)
	assert.False(t, run.CreatedAt.IsZero())
}

func TestRunService_CreateRun_DefaultTrigger(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test")
	run, err := env.runSvc.CreateRun(ctx, sess.ID, "")
	require.NoError(t, err)
	assert.Equal(t, "manual", run.TriggerType)
}

func TestRunService_CreateRun_SessionNotFound(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	_, err := env.runSvc.CreateRun(ctx, "nonexistent", "manual")
	assert.Error(t, err)
}

func TestRunService_GetRunDetail(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Detail Session")
	run, err := env.runSvc.CreateRun(ctx, sess.ID, "manual")
	require.NoError(t, err)

	// Add source fetch (uses run.ID as SessionID field — pre-migration).
	err = env.runSvc.RecordSourceFetch(ctx, &upal.SourceFetch{
		SessionID:  run.ID,
		ToolName:   "hn_fetch",
		SourceType: "static",
		Count:      5,
	})
	require.NoError(t, err)

	// Add analysis.
	err = env.runSvc.RecordAnalysis(ctx, &upal.LLMAnalysis{
		SessionID: run.ID,
		Summary:   "Test analysis",
	})
	require.NoError(t, err)

	// Add workflow runs.
	env.runSvc.SetWorkflowRuns(ctx, run.ID, []upal.WorkflowRun{
		{WorkflowName: "blog-writer", RunID: run.ID, Status: upal.WFRunPending},
	})

	detail, err := env.runSvc.GetRunDetail(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, "Detail Session", detail.SessionName)
	assert.Len(t, detail.Sources, 1)
	assert.NotNil(t, detail.Analysis)
	assert.Equal(t, "Test analysis", detail.Analysis.Summary)
	assert.Len(t, detail.WorkflowRuns, 1)
}

func TestRunService_ApproveReject(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test")

	// Approve
	run1, err := env.runSvc.CreateRun(ctx, sess.ID, "manual")
	require.NoError(t, err)
	err = env.runSvc.ApproveRun(ctx, run1.ID)
	require.NoError(t, err)
	got1, err := env.runSvc.GetRun(ctx, run1.ID)
	require.NoError(t, err)
	assert.Equal(t, upal.SessionRunApproved, got1.Status)
	assert.NotNil(t, got1.ReviewedAt)

	// Reject
	run2, err := env.runSvc.CreateRun(ctx, sess.ID, "manual")
	require.NoError(t, err)
	err = env.runSvc.RejectRun(ctx, run2.ID)
	require.NoError(t, err)
	got2, err := env.runSvc.GetRun(ctx, run2.ID)
	require.NoError(t, err)
	assert.Equal(t, upal.SessionRunRejected, got2.Status)
	assert.NotNil(t, got2.ReviewedAt)
}

func TestRunService_DeleteRun_Cascade(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test")
	run, err := env.runSvc.CreateRun(ctx, sess.ID, "manual")
	require.NoError(t, err)

	// Add children using run.ID as the session-level key (pre-migration).
	env.runSvc.RecordSourceFetch(ctx, &upal.SourceFetch{
		SessionID: run.ID, ToolName: "hn_fetch", SourceType: "static",
	})
	env.runSvc.RecordAnalysis(ctx, &upal.LLMAnalysis{
		SessionID: run.ID, Summary: "test",
	})
	env.runSvc.SetWorkflowRuns(ctx, run.ID, []upal.WorkflowRun{
		{WorkflowName: "test", RunID: run.ID, Status: upal.WFRunPending},
	})
	env.runSvc.RecordPublished(ctx, &upal.PublishedContent{
		SessionID: run.ID, Channel: "youtube",
	})

	err = env.runSvc.DeleteRun(ctx, run.ID)
	require.NoError(t, err)

	// Run should be gone.
	_, err = env.runSvc.GetRun(ctx, run.ID)
	assert.Error(t, err)

	// All children should be cleaned up.
	fetches, _ := env.runSvc.ListSourceFetches(ctx, run.ID)
	assert.Empty(t, fetches)

	wfRuns := env.runSvc.GetWorkflowRuns(ctx, run.ID)
	assert.Empty(t, wfRuns)

	pubs, _ := env.runSvc.ListPublishedByRun(ctx, run.ID)
	assert.Empty(t, pubs)
}

func TestRunService_DeleteRun_NotFound(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	err := env.runSvc.DeleteRun(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestRunService_ListRunsBySession(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sessA := env.createTestSession(t, "Session A")
	sessB := env.createTestSession(t, "Session B")

	// Stagger creation times slightly for stable ordering.
	env.runSvc.CreateRun(ctx, sessA.ID, "manual")
	time.Sleep(time.Millisecond)
	env.runSvc.CreateRun(ctx, sessA.ID, "schedule")
	time.Sleep(time.Millisecond)
	env.runSvc.CreateRun(ctx, sessB.ID, "manual")

	runsA, err := env.runSvc.ListRunsBySession(ctx, sessA.ID)
	require.NoError(t, err)
	assert.Len(t, runsA, 2)
	// Should be newest first.
	assert.True(t, runsA[0].CreatedAt.After(runsA[1].CreatedAt) || runsA[0].CreatedAt.Equal(runsA[1].CreatedAt))

	runsB, err := env.runSvc.ListRunsBySession(ctx, sessB.ID)
	require.NoError(t, err)
	assert.Len(t, runsB, 1)
}

func TestRunService_ListRunsByStatus(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test")
	run1, _ := env.runSvc.CreateRun(ctx, sess.ID, "manual")
	env.runSvc.CreateRun(ctx, sess.ID, "manual")

	env.runSvc.ApproveRun(ctx, run1.ID)

	approved, err := env.runSvc.ListRunsByStatus(ctx, upal.SessionRunApproved)
	require.NoError(t, err)
	assert.Len(t, approved, 1)

	drafts, err := env.runSvc.ListRunsByStatus(ctx, upal.SessionRunDraft)
	require.NoError(t, err)
	assert.Len(t, drafts, 1)
}

func TestRunService_UpdateRunStatus(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test")
	run, _ := env.runSvc.CreateRun(ctx, sess.ID, "manual")

	err := env.runSvc.UpdateRunStatus(ctx, run.ID, upal.SessionRunAnalyzing)
	require.NoError(t, err)

	got, _ := env.runSvc.GetRun(ctx, run.ID)
	assert.Equal(t, upal.SessionRunAnalyzing, got.Status)
}

func TestRunService_UpdateRunSourceCount(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test")
	run, _ := env.runSvc.CreateRun(ctx, sess.ID, "manual")

	err := env.runSvc.UpdateRunSourceCount(ctx, run.ID, 42)
	require.NoError(t, err)

	got, _ := env.runSvc.GetRun(ctx, run.ID)
	assert.Equal(t, 42, got.SourceCount)
}

func TestRunService_DeleteRunsBySession(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test")
	env.runSvc.CreateRun(ctx, sess.ID, "manual")
	env.runSvc.CreateRun(ctx, sess.ID, "schedule")

	err := env.runSvc.DeleteRunsBySession(ctx, sess.ID)
	require.NoError(t, err)

	runs, _ := env.runSvc.ListRunsBySession(ctx, sess.ID)
	assert.Empty(t, runs)
}

func TestRunService_AnalysisMethods(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test")
	run, _ := env.runSvc.CreateRun(ctx, sess.ID, "manual")

	// Record
	err := env.runSvc.RecordAnalysis(ctx, &upal.LLMAnalysis{
		SessionID: run.ID,
		Summary:   "initial",
		SuggestedAngles: []upal.ContentAngle{
			{ID: "angle-1", Headline: "AI Trends", Format: "blog"},
		},
	})
	require.NoError(t, err)

	// Get
	analysis, err := env.runSvc.GetAnalysis(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, "initial", analysis.Summary)

	// Update summary
	err = env.runSvc.UpdateAnalysis(ctx, run.ID, "updated", []string{"insight1"})
	require.NoError(t, err)
	analysis, _ = env.runSvc.GetAnalysis(ctx, run.ID)
	assert.Equal(t, "updated", analysis.Summary)
	assert.Equal(t, []string{"insight1"}, analysis.Insights)

	// Update angles
	err = env.runSvc.UpdateAnalysisAngles(ctx, run.ID, []upal.ContentAngle{
		{ID: "angle-2", Headline: "New Angle", Format: "newsletter"},
	})
	require.NoError(t, err)
	analysis, _ = env.runSvc.GetAnalysis(ctx, run.ID)
	assert.Len(t, analysis.SuggestedAngles, 1)
	assert.Equal(t, "New Angle", analysis.SuggestedAngles[0].Headline)

	// Update angle workflow
	err = env.runSvc.UpdateAngleWorkflow(ctx, run.ID, "angle-2", "blog-writer")
	require.NoError(t, err)
	analysis, _ = env.runSvc.GetAnalysis(ctx, run.ID)
	assert.Equal(t, "blog-writer", analysis.SuggestedAngles[0].WorkflowName)
	assert.Equal(t, "manual", analysis.SuggestedAngles[0].MatchType)
}

func TestRunService_UpdateAngleWorkflow_NotFound(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test")
	run, _ := env.runSvc.CreateRun(ctx, sess.ID, "manual")
	env.runSvc.RecordAnalysis(ctx, &upal.LLMAnalysis{
		SessionID: run.ID, Summary: "test",
	})

	err := env.runSvc.UpdateAngleWorkflow(ctx, run.ID, "nonexistent", "wf")
	assert.Error(t, err)
}

func TestRunService_SurgeMethods(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	err := env.runSvc.CreateSurge(ctx, &upal.SurgeEvent{
		Keyword: "DeepSeek", Multiplier: 10.0,
	})
	require.NoError(t, err)

	surges, _ := env.runSvc.ListSurges(ctx)
	assert.Len(t, surges, 1)

	active, _ := env.runSvc.ListActiveSurges(ctx)
	assert.Len(t, active, 1)

	err = env.runSvc.DismissSurge(ctx, surges[0].ID)
	require.NoError(t, err)

	active, _ = env.runSvc.ListActiveSurges(ctx)
	assert.Empty(t, active)
}

func TestRunService_PublishedMethods(t *testing.T) {
	env := newTestRunEnv()
	ctx := context.Background()

	sess := env.createTestSession(t, "Test")
	run, _ := env.runSvc.CreateRun(ctx, sess.ID, "manual")

	err := env.runSvc.RecordPublished(ctx, &upal.PublishedContent{
		SessionID: run.ID, Channel: "youtube", Title: "Video 1",
	})
	require.NoError(t, err)
	err = env.runSvc.RecordPublished(ctx, &upal.PublishedContent{
		SessionID: run.ID, Channel: "substack", Title: "Post 1",
	})
	require.NoError(t, err)

	all, _ := env.runSvc.ListPublished(ctx)
	assert.Len(t, all, 2)

	byRun, _ := env.runSvc.ListPublishedByRun(ctx, run.ID)
	assert.Len(t, byRun, 2)

	byChannel, _ := env.runSvc.ListPublishedByChannel(ctx, "youtube")
	assert.Len(t, byChannel, 1)
}
