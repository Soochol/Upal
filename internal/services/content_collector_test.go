package services

import (
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestMapPipelineSources_GoogleTrends(t *testing.T) {
	sources := []upal.PipelineSource{
		{ID: "gt1", Type: "google_trends", Keywords: []string{"AI", "LLM"}, Geo: "KR", Limit: 10},
	}
	result := mapPipelineSources(sources, "", false, 0)
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
	result := mapPipelineSources(sources, "", false, 0)
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
	result := mapPipelineSources(sources, "", false, 0)
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
	result := mapPipelineSources(sources, "", false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source (twitter fallback), got %d", len(result))
	}
	if result[0].collectSource.Type != "social" {
		t.Errorf("expected twitter to fallback to social, got %q", result[0].collectSource.Type)
	}
}

func TestConvertToSourceItems_Social(t *testing.T) {
	data := []map[string]any{
		{"title": "AI Topic", "url": "https://bsky.app/1", "content": "AI post", "fetched_from": "bluesky_trending"},
		{"title": "#golang", "url": "https://mastodon.social/tags/golang", "fetched_from": "mastodon_trending_tag"},
	}
	items := convertToSourceItems("social", data)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Title != "AI Topic" {
		t.Errorf("expected title='AI Topic', got %q", items[0].Title)
	}
	if items[0].URL != "https://bsky.app/1" {
		t.Errorf("expected url, got %q", items[0].URL)
	}
	if items[0].Content != "AI post" {
		t.Errorf("expected content='AI post', got %q", items[0].Content)
	}
	if items[0].FetchedFrom != "bluesky_trending" {
		t.Errorf("expected fetched_from='bluesky_trending', got %q", items[0].FetchedFrom)
	}
	if items[1].Title != "#golang" {
		t.Errorf("expected title='#golang', got %q", items[1].Title)
	}
}

func TestConvertToSourceItems_Social_Nil(t *testing.T) {
	items := convertToSourceItems("social", nil)
	if items != nil {
		t.Errorf("expected nil for nil data, got %v", items)
	}
}

func TestConvertToSourceItems_Social_WrongType(t *testing.T) {
	items := convertToSourceItems("social", "not a slice")
	if items != nil {
		t.Errorf("expected nil for wrong type, got %v", items)
	}
}

func TestMapPipelineSources_Research(t *testing.T) {
	sources := []upal.PipelineSource{{
		Type:  "research",
		Topic: "EV market trends",
		Depth: "deep",
	}}
	result := mapPipelineSources(sources, "anthropic/claude-sonnet", false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source, got %d", len(result))
	}
	cs := result[0].collectSource
	if cs.Type != "research" {
		t.Errorf("expected type=research, got %q", cs.Type)
	}
	if cs.Topic != "EV market trends" {
		t.Errorf("expected topic='EV market trends', got %q", cs.Topic)
	}
	if cs.Depth != "deep" {
		t.Errorf("expected depth=deep, got %q", cs.Depth)
	}
	if cs.Model != "anthropic/claude-sonnet" {
		t.Errorf("expected model='anthropic/claude-sonnet', got %q", cs.Model)
	}
}

func TestMapPipelineSources_ResearchDefaultDepth(t *testing.T) {
	sources := []upal.PipelineSource{{
		Type:  "research",
		Topic: "test topic",
	}}
	result := mapPipelineSources(sources, "", false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 mapped source, got %d", len(result))
	}
	if result[0].collectSource.Depth != "light" {
		t.Errorf("expected default depth=light, got %q", result[0].collectSource.Depth)
	}
}

func TestConvertToSourceItems_Research(t *testing.T) {
	data := []map[string]any{
		{"title": "EV Report", "url": "https://example.com", "summary": "Market growing"},
		{"title": "Battery Tech", "url": "https://example.com/battery", "summary": "New advances"},
	}
	items := convertToSourceItems("research", data)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Title != "EV Report" {
		t.Errorf("expected title='EV Report', got %q", items[0].Title)
	}
	if items[0].URL != "https://example.com" {
		t.Errorf("expected url='https://example.com', got %q", items[0].URL)
	}
	if items[0].Content != "Market growing" {
		t.Errorf("expected content='Market growing', got %q", items[0].Content)
	}
	if items[1].Title != "Battery Tech" {
		t.Errorf("expected title='Battery Tech', got %q", items[1].Title)
	}
}

func TestConvertToSourceItems_ResearchNilData(t *testing.T) {
	items := convertToSourceItems("research", nil)
	if items != nil {
		t.Errorf("expected nil for nil data, got %v", items)
	}
}
