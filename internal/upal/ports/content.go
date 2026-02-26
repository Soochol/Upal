package ports

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// ContentSessionPort defines the content session management boundary.
type ContentSessionPort interface {
	// Session CRUD
	CreateSession(ctx context.Context, sess *upal.ContentSession) error
	GetSession(ctx context.Context, id string) (*upal.ContentSession, error)
	ListSessions(ctx context.Context) ([]*upal.ContentSession, error)
	UpdateSession(ctx context.Context, sess *upal.ContentSession) error
	UpdateSessionStatus(ctx context.Context, id string, status upal.ContentSessionStatus) error
	UpdateSessionSettings(ctx context.Context, id string, settings upal.SessionSettings) error
	UpdateSessionSourceCount(ctx context.Context, id string, count int) error
	ApproveSession(ctx context.Context, id string) error
	RejectSession(ctx context.Context, id string) error

	// Session details (composed views joining session + analysis + results)
	GetSessionDetail(ctx context.Context, id string) (*upal.ContentSessionDetail, error)
	ListSessionDetails(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error)
	ListSessionDetailsByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSessionDetail, error)
	ListSessionDetailsByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSessionDetail, error)
	ListSessionDetailsByStatusIncludeArchived(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSessionDetail, error)
	ListAllInstanceSessionDetails(ctx context.Context) ([]*upal.ContentSessionDetail, error)
	ListTemplateDetailsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error)
	ListArchivedSessionDetails(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error)
	ListAllArchivedSessionDetails(ctx context.Context) ([]*upal.ContentSessionDetail, error)

	// Templates
	ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)

	// Archive lifecycle
	ArchiveSession(ctx context.Context, id string) error
	UnarchiveSession(ctx context.Context, id string) error
	DeleteSession(ctx context.Context, id string) error
	DeleteSessionsByPipeline(ctx context.Context, pipelineID string) error

	// Source fetches
	RecordSourceFetch(ctx context.Context, sf *upal.SourceFetch) error
	UpdateSourceFetch(ctx context.Context, sf *upal.SourceFetch) error
	ListSourceFetches(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error)

	// LLM analysis
	RecordAnalysis(ctx context.Context, a *upal.LLMAnalysis) error
	GetAnalysis(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error)
	UpdateAnalysis(ctx context.Context, sessionID string, summary string, insights []string) error
	UpdateAnalysisAngles(ctx context.Context, sessionID string, angles []upal.ContentAngle) error
	UpdateAngleWorkflow(ctx context.Context, sessionID, angleID, workflowName string) error

	// Workflow results (in-memory, no error returns)
	SetWorkflowResults(ctx context.Context, sessionID string, results []upal.WorkflowResult)
	GetWorkflowResults(ctx context.Context, sessionID string) []upal.WorkflowResult

	// Published content
	RecordPublished(ctx context.Context, pc *upal.PublishedContent) error
	ListPublished(ctx context.Context) ([]*upal.PublishedContent, error)
	ListPublishedBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error)
	ListPublishedByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error)

	// Surge events
	CreateSurge(ctx context.Context, se *upal.SurgeEvent) error
	ListSurges(ctx context.Context) ([]*upal.SurgeEvent, error)
	ListActiveSurges(ctx context.Context) ([]*upal.SurgeEvent, error)
	DismissSurge(ctx context.Context, id string) error
}
