package services

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
)

// RetryExecutor wraps WorkflowService.Run with configurable retry and backoff.
type RetryExecutor struct {
	workflowExec  ports.WorkflowExecutor
	runHistorySvc *RunHistoryService
}

// NewRetryExecutor creates a RetryExecutor.
func NewRetryExecutor(workflowExec ports.WorkflowExecutor, runHistorySvc *RunHistoryService) *RetryExecutor {
	return &RetryExecutor{
		workflowExec:  workflowExec,
		runHistorySvc: runHistorySvc,
	}
}

// ExecuteWithRetry runs a workflow with retry on failure.
// It returns channels for events and the final result, same as WorkflowService.Run.
// On retry, previous attempt is marked as failed and a new run record is created.
func (r *RetryExecutor) ExecuteWithRetry(
	ctx context.Context,
	wf *upal.WorkflowDefinition,
	inputs map[string]any,
	policy upal.RetryPolicy,
	triggerType, triggerRef string,
) (<-chan upal.WorkflowEvent, <-chan upal.RunResult, error) {
	outEvents := make(chan upal.WorkflowEvent, 64)
	outResult := make(chan upal.RunResult, 1)

	go func() {
		defer close(outEvents)
		defer close(outResult)

		var firstRunID string

		for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
			// Create run record.
			var retryOf *string
			if attempt > 0 && firstRunID != "" {
				retryOf = &firstRunID
			}

			record, err := r.runHistorySvc.StartRun(ctx, wf.Name, triggerType, triggerRef, inputs)
			if err != nil {
				slog.Warn("retry: failed to create run record", "err", err)
			} else {
				r.runHistorySvc.UpdateRunRetryMeta(ctx, record.ID, attempt, retryOf)
				if attempt == 0 {
					firstRunID = record.ID
				}
			}

			// Execute workflow.
			events, result, execErr := r.workflowExec.Run(ctx, wf, inputs)
			if execErr != nil {
				if record != nil {
					r.runHistorySvc.FailRun(ctx, record.ID, execErr.Error())
				}

				if !isRetryable(execErr) || attempt >= policy.MaxRetries {
					outEvents <- upal.WorkflowEvent{
						Type:    upal.EventError,
						Payload: map[string]any{"error": execErr.Error()},
					}
					return
				}

				sleepWithBackoff(ctx, policy, attempt)
				continue
			}

			// Stream events, watching for errors.
			var hadError bool
			var errMsg string
			for ev := range events {
				outEvents <- ev
				if ev.Type == upal.EventError {
					hadError = true
					errMsg = fmt.Sprintf("%v", ev.Payload["error"])
				}
			}

			res := <-result

			if hadError {
				if record != nil {
					r.runHistorySvc.FailRun(ctx, record.ID, errMsg)
				}

				if !isRetryableMsg(errMsg) || attempt >= policy.MaxRetries {
					return
				}

				sleepWithBackoff(ctx, policy, attempt)
				continue
			}

			// Success.
			if record != nil {
				r.runHistorySvc.CompleteRun(ctx, record.ID, res.State)
			}
			outResult <- res
			return
		}
	}()

	return outEvents, outResult, nil
}

// sleepWithBackoff waits for the backoff duration, respecting context cancellation.
func sleepWithBackoff(ctx context.Context, policy upal.RetryPolicy, attempt int) {
	delay := calculateBackoff(policy, attempt)
	slog.Info("retry: backing off", "attempt", attempt+1, "delay", delay)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}

// calculateBackoff computes the delay for a given attempt using exponential backoff.
func calculateBackoff(policy upal.RetryPolicy, attempt int) time.Duration {
	delay := float64(policy.InitialDelay) * math.Pow(policy.BackoffFactor, float64(attempt))
	if time.Duration(delay) > policy.MaxDelay {
		return policy.MaxDelay
	}
	return time.Duration(delay)
}

// isRetryable checks if an error is worth retrying.
func isRetryable(err error) bool {
	return isRetryableMsg(err.Error())
}

// isRetryableMsg checks if an error message indicates a retryable condition.
func isRetryableMsg(msg string) bool {
	lower := strings.ToLower(msg)
	retryablePatterns := []string{
		"timeout", "rate_limit", "rate limit", "too many requests",
		"429", "500", "502", "503", "504",
		"connection reset", "connection refused", "eof",
		"overloaded", "capacity",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}
