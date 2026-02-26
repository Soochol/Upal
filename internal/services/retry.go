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

var _ ports.RetryExecutor = (*RetryExecutor)(nil)

type RetryExecutor struct {
	workflowExec  ports.WorkflowExecutor
	runHistorySvc ports.RunHistoryPort
}

func NewRetryExecutor(workflowExec ports.WorkflowExecutor, runHistorySvc ports.RunHistoryPort) *RetryExecutor {
	return &RetryExecutor{
		workflowExec:  workflowExec,
		runHistorySvc: runHistorySvc,
	}
}

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
			var retryOf *string
			if attempt > 0 && firstRunID != "" {
				retryOf = &firstRunID
			}

			record, err := r.runHistorySvc.StartRun(ctx, wf.Name, triggerType, triggerRef, inputs, wf)
			if err != nil {
				slog.Warn("retry: failed to create run record", "err", err)
			} else {
				if err := r.runHistorySvc.UpdateRunRetryMeta(ctx, record.ID, attempt, retryOf); err != nil {
					slog.Warn("retry: failed to update retry metadata", "err", err)
				}
				if attempt == 0 {
					firstRunID = record.ID
				}
			}

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

			if record != nil {
				r.runHistorySvc.CompleteRun(ctx, record.ID, res.State)
			}
			outResult <- res
			return
		}
	}()

	return outEvents, outResult, nil
}

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

func calculateBackoff(policy upal.RetryPolicy, attempt int) time.Duration {
	delay := float64(policy.InitialDelay) * math.Pow(policy.BackoffFactor, float64(attempt))
	if time.Duration(delay) > policy.MaxDelay {
		return policy.MaxDelay
	}
	return time.Duration(delay)
}

func isRetryable(err error) bool {
	return isRetryableMsg(err.Error())
}

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
