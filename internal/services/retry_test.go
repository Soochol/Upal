package services

import (
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
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second}, // capped at MaxDelay
		{6, 30 * time.Second}, // still capped
	}

	for _, tt := range tests {
		got := calculateBackoff(policy, tt.attempt)
		if got != tt.expected {
			t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, got)
		}
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		msg       string
		retryable bool
	}{
		{"timeout exceeded", true},
		{"rate_limit_error", true},
		{"Too Many Requests", true},
		{"HTTP 429", true},
		{"HTTP 503 Service Unavailable", true},
		{"connection reset by peer", true},
		{"invalid_api_key", false},
		{"bad request: missing field", false},
		{"node has no model selected", false},
	}

	for _, tt := range tests {
		got := isRetryableMsg(tt.msg)
		if got != tt.retryable {
			t.Errorf("isRetryableMsg(%q) = %v, want %v", tt.msg, got, tt.retryable)
		}
	}
}
