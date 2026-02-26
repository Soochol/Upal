package ports

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// SessionServicePort defines the session management boundary.
type SessionServicePort interface {
	Create(ctx context.Context, s *upal.Session) (*upal.Session, error)
	Get(ctx context.Context, id string) (*upal.Session, error)
	List(ctx context.Context) ([]*upal.Session, error)
	Update(ctx context.Context, s *upal.Session) error
	Delete(ctx context.Context, id string) error
}

// RunServicePort defines the run management boundary.
type RunServicePort interface {
	CreateRun(ctx context.Context, sessionID string, triggerType string) (*upal.Run, error)
	GetRun(ctx context.Context, id string) (*upal.Run, error)
	GetRunDetail(ctx context.Context, id string) (*upal.RunDetail, error)
	ListRuns(ctx context.Context) ([]*upal.RunDetail, error)
	ListRunsBySession(ctx context.Context, sessionID string) ([]*upal.RunDetail, error)
	ListRunsByStatus(ctx context.Context, status upal.SessionRunStatus) ([]*upal.RunDetail, error)
	UpdateRunStatus(ctx context.Context, id string, status upal.SessionRunStatus) error
	ApproveRun(ctx context.Context, id string) error
	RejectRun(ctx context.Context, id string) error
	DeleteRun(ctx context.Context, id string) error

	// Source fetches
	RecordSourceFetch(ctx context.Context, fetch *upal.SourceFetch) error
	UpdateSourceFetch(ctx context.Context, fetch *upal.SourceFetch) error
	ListSourceFetches(ctx context.Context, runID string) ([]*upal.SourceFetch, error)

	// Analysis
	RecordAnalysis(ctx context.Context, analysis *upal.LLMAnalysis) error
	GetAnalysis(ctx context.Context, runID string) (*upal.LLMAnalysis, error)
	UpdateAnalysis(ctx context.Context, runID string, summary string, insights []string) error
	UpdateAnalysisAngles(ctx context.Context, runID string, angles []upal.ContentAngle) error
	UpdateAngleWorkflow(ctx context.Context, runID, angleID, workflowName, workflowRunID string) error

	// Workflow runs
	SetWorkflowRuns(ctx context.Context, runID string, results []upal.WorkflowRun) error
	GetWorkflowRuns(ctx context.Context, runID string) ([]upal.WorkflowRun, error)

	// Published
	RecordPublished(ctx context.Context, p *upal.PublishedContent) error
	ListPublished(ctx context.Context) ([]*upal.PublishedContent, error)
	ListPublishedByRun(ctx context.Context, runID string) ([]*upal.PublishedContent, error)
	ListPublishedByChannel(ctx context.Context, channelID string) ([]*upal.PublishedContent, error)

	// Surge
	CreateSurge(ctx context.Context, event *upal.SurgeEvent) error
	ListSurges(ctx context.Context) ([]*upal.SurgeEvent, error)
	ListActiveSurges(ctx context.Context) ([]*upal.SurgeEvent, error)
	DismissSurge(ctx context.Context, id string) error
}
