package services

import (
	"errors"
	"testing"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func TestCalculateBackoff(t *testing.T) {
	policy := upal.RetryPolicy{
		InitialDelay:  time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"attempt 0 returns InitialDelay", 0, 1 * time.Second},
		{"attempt 1 returns InitialDelay * BackoffFactor", 1, 2 * time.Second},
		{"attempt 2 returns InitialDelay * BackoffFactor^2", 2, 4 * time.Second},
		{"attempt 3 returns InitialDelay * BackoffFactor^3", 3, 8 * time.Second},
		{"attempt 4 returns InitialDelay * BackoffFactor^4", 4, 16 * time.Second},
		{"attempt 5 capped at MaxDelay", 5, 30 * time.Second},
		{"attempt 6 still capped at MaxDelay", 6, 30 * time.Second},
		{"attempt 10 still capped at MaxDelay", 10, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBackoff(policy, tt.attempt)
			if got != tt.expected {
				t.Errorf("calculateBackoff(policy, %d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}

func TestCalculateBackoff_CustomPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policy   upal.RetryPolicy
		attempt  int
		expected time.Duration
	}{
		{
			name: "factor 3 attempt 0",
			policy: upal.RetryPolicy{
				InitialDelay:  500 * time.Millisecond,
				MaxDelay:      1 * time.Minute,
				BackoffFactor: 3.0,
			},
			attempt:  0,
			expected: 500 * time.Millisecond,
		},
		{
			name: "factor 3 attempt 1",
			policy: upal.RetryPolicy{
				InitialDelay:  500 * time.Millisecond,
				MaxDelay:      1 * time.Minute,
				BackoffFactor: 3.0,
			},
			attempt:  1,
			expected: 1500 * time.Millisecond,
		},
		{
			name: "factor 3 attempt 2",
			policy: upal.RetryPolicy{
				InitialDelay:  500 * time.Millisecond,
				MaxDelay:      1 * time.Minute,
				BackoffFactor: 3.0,
			},
			attempt:  2,
			expected: 4500 * time.Millisecond,
		},
		{
			name: "factor 3 attempt 3",
			policy: upal.RetryPolicy{
				InitialDelay:  500 * time.Millisecond,
				MaxDelay:      1 * time.Minute,
				BackoffFactor: 3.0,
			},
			attempt:  3,
			expected: 13500 * time.Millisecond,
		},
		{
			name: "factor 3 capped at MaxDelay",
			policy: upal.RetryPolicy{
				InitialDelay:  500 * time.Millisecond,
				MaxDelay:      1 * time.Minute,
				BackoffFactor: 3.0,
			},
			attempt:  10,
			expected: 1 * time.Minute,
		},
		{
			name: "factor 1.5 attempt 2",
			policy: upal.RetryPolicy{
				InitialDelay:  2 * time.Second,
				MaxDelay:      10 * time.Second,
				BackoffFactor: 1.5,
			},
			attempt:  2,
			expected: time.Duration(float64(2*time.Second) * 1.5 * 1.5),
		},
		{
			name: "large initial delay capped immediately",
			policy: upal.RetryPolicy{
				InitialDelay:  10 * time.Second,
				MaxDelay:      5 * time.Second,
				BackoffFactor: 2.0,
			},
			attempt:  0,
			expected: 5 * time.Second,
		},
		{
			name: "factor 1 always returns InitialDelay",
			policy: upal.RetryPolicy{
				InitialDelay:  3 * time.Second,
				MaxDelay:      1 * time.Minute,
				BackoffFactor: 1.0,
			},
			attempt:  5,
			expected: 3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBackoff(tt.policy, tt.attempt)
			if got != tt.expected {
				t.Errorf("calculateBackoff(%+v, %d) = %v, want %v", tt.policy, tt.attempt, got, tt.expected)
			}
		})
	}
}

func TestCalculateBackoff_DefaultPolicy(t *testing.T) {
	policy := upal.DefaultRetryPolicy()

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"default attempt 0", 0, 1 * time.Second},
		{"default attempt 1", 1, 2 * time.Second},
		{"default attempt 2", 2, 4 * time.Second},
		{"default attempt 3", 3, 8 * time.Second},
		{"default attempt 8", 8, 256 * time.Second},
		{"default attempt 9 capped at 5m", 9, 5 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBackoff(policy, tt.attempt)
			if got != tt.expected {
				t.Errorf("calculateBackoff(DefaultRetryPolicy(), %d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}

func TestIsRetryableMsg(t *testing.T) {
	tests := []struct {
		name      string
		msg       string
		retryable bool
	}{
		// Retryable patterns.
		{"timeout lowercase", "timeout", true},
		{"timeout in sentence", "request timeout exceeded", true},
		{"timeout uppercase", "TIMEOUT", true},
		{"rate_limit underscore", "rate_limit_error", true},
		{"rate limit with space", "rate limit exceeded", true},
		{"too many requests", "too many requests", true},
		{"429 status code", "429 Too Many Requests", true},
		{"429 in message", "HTTP 429", true},
		{"500 status code", "500 Internal Server Error", true},
		{"502 status code", "502 Bad Gateway", true},
		{"503 status code", "503 Service Unavailable", true},
		{"504 status code", "504 Gateway Timeout", true},
		{"connection reset", "connection reset by peer", true},
		{"connection refused", "connection refused", true},
		{"eof lowercase", "eof", true},
		{"eof uppercase", "unexpected EOF", true},
		{"overloaded", "overloaded", true},
		{"overloaded in sentence", "server is overloaded, try again later", true},
		{"capacity", "capacity", true},
		{"capacity in sentence", "insufficient capacity to process request", true},
		{"mixed case retryable", "Connection Reset By Peer", true},

		// Non-retryable patterns.
		{"invalid input", "invalid input", false},
		{"not found", "not found", false},
		{"permission denied", "permission denied", false},
		{"empty string", "", false},
		{"bad request", "bad request: missing field", false},
		{"invalid api key", "invalid_api_key", false},
		{"no model selected", "node has no model selected", false},
		{"authentication error", "authentication failed", false},
		{"validation error", "validation error: field required", false},
		{"unknown provider", "unknown provider: foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableMsg(tt.msg)
			if got != tt.retryable {
				t.Errorf("isRetryableMsg(%q) = %v, want %v", tt.msg, got, tt.retryable)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"timeout error", errors.New("timeout"), true},
		{"rate limit error", errors.New("rate limit exceeded"), true},
		{"429 error", errors.New("HTTP 429 Too Many Requests"), true},
		{"503 error", errors.New("503 Service Unavailable"), true},
		{"connection reset error", errors.New("connection reset by peer"), true},
		{"connection refused error", errors.New("connection refused"), true},
		{"eof error", errors.New("unexpected EOF"), true},
		{"overloaded error", errors.New("server overloaded"), true},
		{"capacity error", errors.New("insufficient capacity"), true},
		{"bad request error", errors.New("bad request"), false},
		{"not found error", errors.New("workflow not found"), false},
		{"permission error", errors.New("permission denied"), false},
		{"generic error", errors.New("something went wrong"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryable(tt.err)
			if got != tt.retryable {
				t.Errorf("isRetryable(%v) = %v, want %v", tt.err, got, tt.retryable)
			}
		})
	}
}

func TestIsRetryableMsg_PatternBoundaries(t *testing.T) {
	// Verify that patterns match as substrings, not just exact matches.
	tests := []struct {
		name      string
		msg       string
		retryable bool
	}{
		{"429 embedded in URL", "POST https://api.example.com returned 429", true},
		{"timeout as prefix", "timeout: context deadline exceeded", true},
		{"eof at end", "read: unexpected eof", true},
		{"rate_limit in JSON", `{"error": "rate_limit_exceeded"}`, true},
		{"502 in response body", "upstream returned 502 error", true},
		{"504 with details", "504 gateway timeout after 30s", true},
		{"overloaded with prefix", "model is currently overloaded", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableMsg(tt.msg)
			if got != tt.retryable {
				t.Errorf("isRetryableMsg(%q) = %v, want %v", tt.msg, got, tt.retryable)
			}
		})
	}
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := upal.DefaultRetryPolicy()

	if policy.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", policy.MaxRetries)
	}
	if policy.InitialDelay != time.Second {
		t.Errorf("InitialDelay = %v, want %v", policy.InitialDelay, time.Second)
	}
	if policy.MaxDelay != 5*time.Minute {
		t.Errorf("MaxDelay = %v, want %v", policy.MaxDelay, 5*time.Minute)
	}
	if policy.BackoffFactor != 2.0 {
		t.Errorf("BackoffFactor = %f, want 2.0", policy.BackoffFactor)
	}
}

func TestNewRetryExecutor(t *testing.T) {
	executor := NewRetryExecutor(nil, nil)
	if executor == nil {
		t.Fatal("NewRetryExecutor returned nil")
	}
	if executor.workflowExec != nil {
		t.Error("expected workflowExec to be nil")
	}
	if executor.runHistorySvc != nil {
		t.Error("expected runHistorySvc to be nil")
	}
}
