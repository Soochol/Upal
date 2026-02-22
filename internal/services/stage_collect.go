package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
	"github.com/soochol/upal/internal/upal"
	"golang.org/x/sync/errgroup"
)

// CollectStageExecutor fetches data from external sources (RSS, HTTP, web scrape)
// without LLM involvement. The collected text is passed to subsequent stages via
// the stage output "text" field.
type CollectStageExecutor struct {
	httpClient *http.Client
}

func NewCollectStageExecutor() *CollectStageExecutor {
	return &CollectStageExecutor{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *CollectStageExecutor) Type() string { return "collect" }

func (e *CollectStageExecutor) Execute(ctx context.Context, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	now := time.Now()
	completedAt := now

	sources := stage.Config.Sources
	if len(sources) == 0 {
		completedAt = time.Now()
		return &upal.StageResult{
			StageID:     stage.ID,
			Status:      "completed",
			Output:      map[string]any{"text": "", "sources": map[string]any{}},
			StartedAt:   now,
			CompletedAt: &completedAt,
		}, nil
	}

	type sourceResult struct {
		id   string
		text string
		data any
	}

	results := make([]sourceResult, len(sources))
	g, gCtx := errgroup.WithContext(ctx)

	for i, src := range sources {
		g.Go(func() error {
			text, data, err := e.fetchSource(gCtx, src)
			if err != nil {
				// Partial failure: include error in text, don't abort
				results[i] = sourceResult{
					id:   src.ID,
					text: fmt.Sprintf("=== %s: %s ===\n[오류] %v\n", strings.ToUpper(src.Type), src.ID, err),
					data: map[string]any{"error": err.Error()},
				}
				return nil
			}
			results[i] = sourceResult{id: src.ID, text: text, data: data}
			return nil
		})
	}

	_ = g.Wait() // errors are embedded in results, not returned

	var textParts []string
	sourcesMap := make(map[string]any)
	for _, r := range results {
		if r.id == "" {
			continue
		}
		textParts = append(textParts, r.text)
		sourcesMap[r.id] = r.data
	}

	combinedText := strings.Join(textParts, "\n")
	completedAt = time.Now()

	return &upal.StageResult{
		StageID: stage.ID,
		Status:  "completed",
		Output: map[string]any{
			"text":    combinedText,
			"sources": sourcesMap,
		},
		StartedAt:   now,
		CompletedAt: &completedAt,
	}, nil
}

// fetchSource dispatches to the appropriate handler based on source type.
func (e *CollectStageExecutor) fetchSource(ctx context.Context, src upal.CollectSource) (text string, data any, err error) {
	switch src.Type {
	case "rss":
		return e.fetchRSS(ctx, src)
	case "http":
		return e.fetchHTTP(ctx, src)
	case "scrape":
		return e.fetchScrape(ctx, src)
	default:
		return "", nil, fmt.Errorf("unknown source type %q", src.Type)
	}
}

func (e *CollectStageExecutor) fetchRSS(ctx context.Context, src upal.CollectSource) (string, any, error) {
	fp := gofeed.NewParser()
	fp.Client = e.httpClient

	feed, err := fp.ParseURLWithContext(src.URL, ctx)
	if err != nil {
		return "", nil, fmt.Errorf("RSS parse failed: %w", err)
	}

	limit := src.Limit
	if limit <= 0 {
		limit = 20
	}

	var items []map[string]any
	var sb strings.Builder
	fmt.Fprintf(&sb, "=== RSS: %s ===\n피드: %s\n\n", src.ID, feed.Title)

	for _, item := range feed.Items {
		if len(items) >= limit {
			break
		}
		published := ""
		if item.PublishedParsed != nil {
			published = item.PublishedParsed.Format("2006-01-02")
		} else if item.Published != "" {
			published = item.Published
		}
		desc := item.Description
		if len(desc) > 300 {
			desc = desc[:300] + "…"
		}
		items = append(items, map[string]any{
			"title":       item.Title,
			"link":        item.Link,
			"published":   published,
			"description": desc,
		})
		if published != "" {
			fmt.Fprintf(&sb, "[%s] %s\n%s\n%s\n\n", published, item.Title, item.Link, desc)
		} else {
			fmt.Fprintf(&sb, "%s\n%s\n%s\n\n", item.Title, item.Link, desc)
		}
	}

	return sb.String(), items, nil
}

func (e *CollectStageExecutor) fetchHTTP(ctx context.Context, src upal.CollectSource) (string, any, error) {
	method := src.Method
	if method == "" {
		method = "GET"
	}

	var bodyReader io.Reader
	if src.Body != "" {
		bodyReader = strings.NewReader(src.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, src.URL, bodyReader)
	if err != nil {
		return "", nil, fmt.Errorf("request build failed: %w", err)
	}
	for k, v := range src.Headers {
		req.Header.Set(k, v)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return "", nil, fmt.Errorf("response read failed: %w", err)
	}
	bodyStr := string(bodyBytes)

	// Try to parse as JSON for structured output
	var parsed any
	if json.Unmarshal(bodyBytes, &parsed) == nil {
		// Pretty-print for LLM consumption
		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		bodyStr = string(pretty)
	}

	text := fmt.Sprintf("=== HTTP: %s ===\n상태: %d\n\n%s\n", src.ID, resp.StatusCode, bodyStr)
	data := map[string]any{
		"status": resp.StatusCode,
		"body":   bodyStr,
	}
	return text, data, nil
}

func (e *CollectStageExecutor) fetchScrape(ctx context.Context, src upal.CollectSource) (string, any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", src.URL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("request build failed: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; UpalBot/1.0)")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("HTML parse failed: %w", err)
	}

	limit := src.ScrapeLimit
	if limit <= 0 {
		limit = 30
	}
	attr := src.Attribute

	var items []string
	sel := src.Selector
	if sel == "" {
		sel = "body"
	}

	doc.Find(sel).EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i >= limit {
			return false
		}
		var val string
		if attr != "" {
			val, _ = s.Attr(attr)
		} else {
			val = strings.TrimSpace(s.Text())
		}
		if val != "" {
			items = append(items, val)
		}
		return true
	})

	var sb strings.Builder
	fmt.Fprintf(&sb, "=== Scrape: %s ===\nURL: %s\n\n", src.ID, src.URL)
	for _, item := range items {
		fmt.Fprintln(&sb, item)
	}

	return sb.String(), items, nil
}
