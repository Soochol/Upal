package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

type ContentSessionRepository interface {
	Create(ctx context.Context, s *upal.ContentSession) error
	Get(ctx context.Context, id string) (*upal.ContentSession, error)
	List(ctx context.Context) ([]*upal.ContentSession, error)
	ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	ListByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error)
	ListAllByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error)
	ListByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSession, error)
	ListArchivedByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	ListArchived(ctx context.Context) ([]*upal.ContentSession, error)
	ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	Update(ctx context.Context, s *upal.ContentSession) error
	Delete(ctx context.Context, id string) error
}

type SourceFetchRepository interface {
	Create(ctx context.Context, sf *upal.SourceFetch) error
	Update(ctx context.Context, sf *upal.SourceFetch) error
	ListBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error)
}

type LLMAnalysisRepository interface {
	Create(ctx context.Context, a *upal.LLMAnalysis) error
	GetBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error)
	Update(ctx context.Context, a *upal.LLMAnalysis) error
}

type PublishedContentRepository interface {
	Create(ctx context.Context, pc *upal.PublishedContent) error
	List(ctx context.Context) ([]*upal.PublishedContent, error)
	ListBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error)
	ListByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error)
	DeleteBySession(ctx context.Context, sessionID string) error
}

type WorkflowResultRepository interface {
	Save(ctx context.Context, sessionID string, results []upal.WorkflowResult) error
	GetBySession(ctx context.Context, sessionID string) ([]upal.WorkflowResult, error)
	DeleteBySession(ctx context.Context, sessionID string) error
}

type SurgeEventRepository interface {
	Create(ctx context.Context, se *upal.SurgeEvent) error
	List(ctx context.Context) ([]*upal.SurgeEvent, error)
	ListActive(ctx context.Context) ([]*upal.SurgeEvent, error)
	Get(ctx context.Context, id string) (*upal.SurgeEvent, error)
	Update(ctx context.Context, se *upal.SurgeEvent) error
}
