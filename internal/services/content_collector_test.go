package services

import (
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestMapPipelineSources_GoogleTrends(t *testing.T) {
	sources := []upal.PipelineSource{
		{ID: "gt1", Type: "google_trends", Keywords: []string{"AI", "LLM"}, Geo: "KR", Limit: 10},
	}
	result := mapPipelineSources(sources, false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source, got %d", len(result))
	}
	cs := result[0].collectSource
	if cs.Type != "rss" {
		t.Errorf("expected type=rss, got %q", cs.Type)
	}
	if cs.URL != "https://trends.google.com/trending/rss?geo=KR" {
		t.Errorf("unexpected URL: %s", cs.URL)
	}
	if cs.Limit != 10 {
		t.Errorf("expected limit=10, got %d", cs.Limit)
	}
}

func TestMapPipelineSources_GoogleTrends_DefaultGeo(t *testing.T) {
	sources := []upal.PipelineSource{
		{ID: "gt2", Type: "google_trends"},
	}
	result := mapPipelineSources(sources, false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source, got %d", len(result))
	}
	if result[0].collectSource.URL != "https://trends.google.com/trending/rss?geo=US" {
		t.Errorf("expected default geo=US, got URL: %s", result[0].collectSource.URL)
	}
}

func TestMapPipelineSources_Social(t *testing.T) {
	sources := []upal.PipelineSource{
		{ID: "soc1", Type: "social", Keywords: []string{"AI", "startup"}, Limit: 15},
	}
	result := mapPipelineSources(sources, false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source, got %d", len(result))
	}
	cs := result[0].collectSource
	if cs.Type != "social" {
		t.Errorf("expected type=social, got %q", cs.Type)
	}
	if len(cs.Keywords) != 2 || cs.Keywords[0] != "AI" {
		t.Errorf("expected keywords=[AI startup], got %v", cs.Keywords)
	}
}

func TestMapPipelineSources_TwitterFallback(t *testing.T) {
	sources := []upal.PipelineSource{
		{ID: "tw1", Type: "twitter", Keywords: []string{"Go"}},
	}
	result := mapPipelineSources(sources, false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source (twitter fallback), got %d", len(result))
	}
	if result[0].collectSource.Type != "social" {
		t.Errorf("expected twitter to fallback to social, got %q", result[0].collectSource.Type)
	}
}
