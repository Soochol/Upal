package agents

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func TestSensorNodeBuilder_Build(t *testing.T) {
	builder := &SensorNodeBuilder{}
	if builder.NodeType() != upal.NodeTypeSensor {
		t.Fatalf("NodeType() = %q, want %q", builder.NodeType(), upal.NodeTypeSensor)
	}

	nd := &upal.NodeDefinition{
		ID:   "sensor1",
		Type: upal.NodeTypeSensor,
		Config: map[string]any{
			"mode":     "poll",
			"url":      "http://example.com/status",
			"interval": float64(5),
			"timeout":  float64(30),
		},
	}

	ag, err := builder.Build(nd, BuildDeps{})
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if ag == nil {
		t.Fatal("Build() returned nil agent")
	}
}

func TestSensorNodeBuilder_WebhookMode(t *testing.T) {
	builder := &SensorNodeBuilder{}

	nd := &upal.NodeDefinition{
		ID:   "sensor-wh",
		Type: upal.NodeTypeSensor,
		Config: map[string]any{
			"mode":    "webhook",
			"timeout": float64(10),
		},
	}

	ag, err := builder.Build(nd, BuildDeps{})
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if ag == nil {
		t.Fatal("Build() returned nil agent")
	}
}

func TestSensorNodeBuilder_DefaultConfig(t *testing.T) {
	builder := &SensorNodeBuilder{}

	nd := &upal.NodeDefinition{
		ID:     "sensor-default",
		Type:   upal.NodeTypeSensor,
		Config: map[string]any{},
	}

	ag, err := builder.Build(nd, BuildDeps{})
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if ag == nil {
		t.Fatal("Build() returned nil agent")
	}
}

func TestHttpGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	body, err := httpGet(context.Background(), client, srv.URL)
	if err != nil {
		t.Fatalf("httpGet error: %v", err)
	}
	if body != `{"status":"ready"}` {
		t.Errorf("body = %q, want %q", body, `{"status":"ready"}`)
	}
}

func TestHttpGet_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	_, err := httpGet(context.Background(), client, srv.URL)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
