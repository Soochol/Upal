package services

import (
	"context"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// SessionService manages the lifecycle of Session entities.
type SessionService struct {
	repo repository.SessionRepository
}

// NewSessionService creates a new SessionService.
func NewSessionService(repo repository.SessionRepository) *SessionService {
	return &SessionService{repo: repo}
}

// Create validates and persists a new Session.
func (s *SessionService) Create(ctx context.Context, sess *upal.Session) (*upal.Session, error) {
	if sess.Name == "" {
		return nil, fmt.Errorf("session name is required")
	}
	if sess.ID == "" {
		sess.ID = upal.GenerateID("sess")
	}
	if sess.Status == "" {
		sess.Status = upal.SessionStatusDraft
	}
	for i := range sess.Stages {
		if sess.Stages[i].ID == "" {
			sess.Stages[i].ID = upal.GenerateID("stg")
		}
	}
	now := time.Now()
	sess.CreatedAt = now
	sess.UpdatedAt = now
	if err := s.repo.Create(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

// Get retrieves a Session by ID.
func (s *SessionService) Get(ctx context.Context, id string) (*upal.Session, error) {
	return s.repo.Get(ctx, id)
}

// List returns all Sessions.
func (s *SessionService) List(ctx context.Context) ([]*upal.Session, error) {
	return s.repo.List(ctx)
}

// Update persists changes to an existing Session.
func (s *SessionService) Update(ctx context.Context, sess *upal.Session) error {
	sess.UpdatedAt = time.Now()
	return s.repo.Update(ctx, sess)
}

// Delete removes a Session by ID.
func (s *SessionService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// UpdateSettings applies partial updates to a Session that is in draft or active status.
func (s *SessionService) UpdateSettings(ctx context.Context, id string, settings upal.SessionSettings) error {
	sess, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if sess.Status != upal.SessionStatusDraft && sess.Status != upal.SessionStatusActive {
		return fmt.Errorf("session %q: settings can only be changed in draft or active status", id)
	}
	if settings.Name != "" {
		sess.Name = settings.Name
	}
	if settings.Sources != nil {
		sess.Sources = toSessionSources(settings.Sources)
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
		sess.Workflows = toSessionWorkflows(settings.Workflows)
	}
	if settings.Context != nil {
		sess.Context = toSessionContext(settings.Context)
	}
	sess.UpdatedAt = time.Now()
	return s.repo.Update(ctx, sess)
}

// toSessionSources converts PipelineSource (old type) to SessionSource (new type).
func toSessionSources(ps []upal.PipelineSource) []upal.SessionSource {
	out := make([]upal.SessionSource, len(ps))
	for i, p := range ps {
		out[i] = upal.SessionSource{
			ID:         p.ID,
			PipelineID: p.PipelineID,
			ToolName:   p.ToolName,
			SourceType: p.SourceType,
			Config:     p.Config,
			Enabled:    p.Enabled,
			Type:       p.Type,
			Label:      p.Label,
			URL:        p.URL,
			Subreddit:  p.Subreddit,
			MinScore:   p.MinScore,
			Keywords:   p.Keywords,
			Accounts:   p.Accounts,
			Geo:        p.Geo,
			Limit:      p.Limit,
			Topic:      p.Topic,
			Depth:      p.Depth,
			Model:      p.Model,
		}
	}
	return out
}

// toSessionWorkflows converts PipelineWorkflow (old type) to SessionWorkflow (new type).
func toSessionWorkflows(pw []upal.PipelineWorkflow) []upal.SessionWorkflow {
	out := make([]upal.SessionWorkflow, len(pw))
	for i, w := range pw {
		out[i] = upal.SessionWorkflow{
			WorkflowName: w.WorkflowName,
			Label:        w.Label,
			AutoSelect:   w.AutoSelect,
			ChannelID:    w.ChannelID,
		}
	}
	return out
}

// toSessionContext converts PipelineContext (old type) to SessionContext (new type).
func toSessionContext(pc *upal.PipelineContext) *upal.SessionContext {
	if pc == nil {
		return nil
	}
	return &upal.SessionContext{
		Prompt:        pc.Prompt,
		Language:      pc.Language,
		ResearchDepth: pc.ResearchDepth,
		ResearchModel: pc.ResearchModel,
	}
}
