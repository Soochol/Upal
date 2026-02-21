package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// RunHistoryService manages workflow execution records.
type RunHistoryService struct {
	runRepo repository.RunRepository
}

// NewRunHistoryService creates a RunHistoryService.
func NewRunHistoryService(runRepo repository.RunRepository) *RunHistoryService {
	return &RunHistoryService{runRepo: runRepo}
}

// StartRun creates a new RunRecord in pending/running state.
func (s *RunHistoryService) StartRun(ctx context.Context, workflowName, triggerType, triggerRef string, inputs map[string]any) (*upal.RunRecord, error) {
	now := time.Now()
	record := &upal.RunRecord{
		ID:           upal.GenerateID("run"),
		WorkflowName: workflowName,
		TriggerType:  triggerType,
		TriggerRef:   triggerRef,
		Status:       upal.RunStatusRunning,
		Inputs:       inputs,
		CreatedAt:    now,
		StartedAt:    &now,
	}

	if err := s.runRepo.Create(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

// CompleteRun marks a run as successful with outputs.
func (s *RunHistoryService) CompleteRun(ctx context.Context, id string, outputs map[string]any) error {
	record, err := s.runRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	record.Status = upal.RunStatusSuccess
	record.Outputs = outputs
	record.CompletedAt = &now
	return s.runRepo.Update(ctx, record)
}

// FailRun marks a run as failed with an error message.
func (s *RunHistoryService) FailRun(ctx context.Context, id string, errMsg string) error {
	record, err := s.runRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	record.Status = upal.RunStatusFailed
	record.Error = &errMsg
	record.CompletedAt = &now
	return s.runRepo.Update(ctx, record)
}

// UpdateNodeRun adds or updates a node run record within a run.
func (s *RunHistoryService) UpdateNodeRun(ctx context.Context, runID string, nodeRun upal.NodeRunRecord) error {
	record, err := s.runRepo.Get(ctx, runID)
	if err != nil {
		return err
	}

	// Find existing node run or append new one.
	found := false
	for i, nr := range record.NodeRuns {
		if nr.NodeID == nodeRun.NodeID {
			record.NodeRuns[i] = nodeRun
			found = true
			break
		}
	}
	if !found {
		record.NodeRuns = append(record.NodeRuns, nodeRun)
	}

	return s.runRepo.Update(ctx, record)
}

// GetRun retrieves a single run record.
func (s *RunHistoryService) GetRun(ctx context.Context, id string) (*upal.RunRecord, error) {
	return s.runRepo.Get(ctx, id)
}

// ListRuns returns runs for a specific workflow with pagination.
func (s *RunHistoryService) ListRuns(ctx context.Context, workflowName string, limit, offset int) ([]*upal.RunRecord, int, error) {
	return s.runRepo.ListByWorkflow(ctx, workflowName, limit, offset)
}

// ListAllRuns returns all runs with pagination. status filters by run status when non-empty.
func (s *RunHistoryService) ListAllRuns(ctx context.Context, limit, offset int, status string) ([]*upal.RunRecord, int, error) {
	return s.runRepo.ListAll(ctx, limit, offset, status)
}

// CleanupOrphanedRuns marks all running/pending runs as failed.
// Should be called once at server startup.
func (s *RunHistoryService) CleanupOrphanedRuns(ctx context.Context) {
	type orphanCleaner interface {
		MarkOrphanedRunsFailed(ctx context.Context) (int64, error)
	}
	if c, ok := s.runRepo.(orphanCleaner); ok {
		n, err := c.MarkOrphanedRunsFailed(ctx)
		if err != nil {
			slog.Warn("failed to clean up orphaned runs", "err", err)
			return
		}
		if n > 0 {
			slog.Info("marked orphaned runs as failed", "count", n)
		}
	}
}
