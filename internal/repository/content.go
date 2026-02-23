package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// ContentSessionRepository manages ContentSession persistence.
type ContentSessionRepository interface {
	Create(ctx context.Context, s *upal.ContentSession) error
	Get(ctx context.Context, id string) (*upal.ContentSession, error)
	List(ctx context.Context) ([]*upal.ContentSession, error)
	ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	ListByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error)
	ListByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSession, error)
	Update(ctx context.Context, s *upal.ContentSession) error
}

// SourceFetchRepository manages SourceFetch persistence.
type SourceFetchRepository interface {
	Create(ctx context.Context, sf *upal.SourceFetch) error
	ListBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error)
}

// LLMAnalysisRepository manages LLMAnalysis persistence.
type LLMAnalysisRepository interface {
	Create(ctx context.Context, a *upal.LLMAnalysis) error
	GetBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error)
	Update(ctx context.Context, a *upal.LLMAnalysis) error
}

// PublishedContentRepository manages PublishedContent persistence.
type PublishedContentRepository interface {
	Create(ctx context.Context, pc *upal.PublishedContent) error
	List(ctx context.Context) ([]*upal.PublishedContent, error)
	ListBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error)
	ListByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error)
}

// SurgeEventRepository manages SurgeEvent persistence.
type SurgeEventRepository interface {
	Create(ctx context.Context, se *upal.SurgeEvent) error
	List(ctx context.Context) ([]*upal.SurgeEvent, error)
	ListActive(ctx context.Context) ([]*upal.SurgeEvent, error)
	Get(ctx context.Context, id string) (*upal.SurgeEvent, error)
	Update(ctx context.Context, se *upal.SurgeEvent) error
}
