package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/soochol/upal/internal/llmutil"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// ContentCollector orchestrates content collection, LLM analysis, and
// workflow production for content pipelines. It bridges the CollectStageExecutor
// (source fetching) with the ContentSessionService (recording) and WorkflowService
// (production).
type ContentCollector struct {
	contentSvc   *ContentSessionService
	collectExec  *CollectStageExecutor
	workflowSvc  *WorkflowService
	workflowRepo repository.WorkflowRepository
	resolver     ports.LLMResolver
}

// NewContentCollector creates a ContentCollector with all required dependencies.
func NewContentCollector(
	contentSvc *ContentSessionService,
	collectExec *CollectStageExecutor,
	workflowSvc *WorkflowService,
	workflowRepo repository.WorkflowRepository,
	resolver ports.LLMResolver,
) *ContentCollector {
	return &ContentCollector{
		contentSvc:   contentSvc,
		collectExec:  collectExec,
		workflowSvc:  workflowSvc,
		workflowRepo: workflowRepo,
		resolver:     resolver,
	}
}

// CollectAndAnalyze fetches content from pipeline sources, records the results,
// runs LLM analysis on the collected data, and transitions the session to
// pending_review. This is designed to run in a background goroutine.
//
// If isTest is true, each source's item limit is overridden with the provided
// limit value for faster iteration.
func (c *ContentCollector) CollectAndAnalyze(ctx context.Context, pipeline *upal.Pipeline, session *upal.ContentSession, isTest bool, limit int) {
	// Map pipeline sources to collect sources.
	sources := mapPipelineSources(pipeline.Sources, isTest, limit)

	if len(sources) == 0 {
		log.Printf("content_collector: no fetchable sources for pipeline %s", pipeline.ID)
		if err := c.contentSvc.UpdateSessionStatus(ctx, session.ID, upal.SessionPendingReview); err != nil {
			log.Printf("content_collector: failed to update session status: %v", err)
		}
		return
	}

	// Fetch each source and record results.
	totalItems := 0
	for _, mapped := range sources {
		pipelineSrc := pipeline.Sources[mapped.pipelineIndex]
		sf := c.fetchAndRecord(ctx, session.ID, pipelineSrc, mapped.collectSource)
		if sf != nil && sf.Error == nil {
			totalItems += len(sf.RawItems)
		}
	}

	// Persist source count and transition to analyzing.
	if err := c.contentSvc.UpdateSessionSourceCount(ctx, session.ID, totalItems); err != nil {
		log.Printf("content_collector: failed to update source count: %v", err)
	}
	if err := c.contentSvc.UpdateSessionStatus(ctx, session.ID, upal.SessionAnalyzing); err != nil {
		log.Printf("content_collector: failed to transition to analyzing: %v", err)
	}

	// Run LLM analysis if we collected any items.
	if totalItems > 0 {
		c.runAnalysis(ctx, pipeline, session)
	}

	// Transition to pending_review regardless of analysis outcome.
	if err := c.contentSvc.UpdateSessionStatus(ctx, session.ID, upal.SessionPendingReview); err != nil {
		log.Printf("content_collector: failed to transition to pending_review: %v", err)
	}
}

// fetchAndRecord calls the appropriate fetcher for a source and records
// the SourceFetch result. Returns the recorded SourceFetch (may have Error set).
func (c *ContentCollector) fetchAndRecord(ctx context.Context, sessionID string, pipelineSrc upal.PipelineSource, src upal.CollectSource) *upal.SourceFetch {
	sf := &upal.SourceFetch{
		SessionID:  sessionID,
		ToolName:   pipelineSrc.Type,
		SourceType: pipelineSrc.SourceType,
		Label:      pipelineSrc.Label,
	}

	fetcher, ok := c.collectExec.Fetcher(src.Type)
	if !ok {
		errMsg := fmt.Sprintf("no fetcher for source type %q", src.Type)
		sf.Error = &errMsg
		if err := c.contentSvc.RecordSourceFetch(ctx, sf); err != nil {
			log.Printf("content_collector: failed to record source fetch error: %v", err)
		}
		return sf
	}

	text, data, err := fetcher.Fetch(ctx, src)
	if err != nil {
		errMsg := err.Error()
		sf.Error = &errMsg
		if recErr := c.contentSvc.RecordSourceFetch(ctx, sf); recErr != nil {
			log.Printf("content_collector: failed to record source fetch error: %v", recErr)
		}
		return sf
	}

	// Convert fetcher data to SourceItems.
	sf.RawItems = convertToSourceItems(src.Type, data)
	sf.Count = len(sf.RawItems)
	_ = text // text is used for prompt building via ListSourceFetches

	if err := c.contentSvc.RecordSourceFetch(ctx, sf); err != nil {
		log.Printf("content_collector: failed to record source fetch: %v", err)
	}
	return sf
}

// convertToSourceItems converts fetcher output data into a slice of SourceItem.
func convertToSourceItems(sourceType string, data any) []upal.SourceItem {
	if data == nil {
		return nil
	}

	switch sourceType {
	case "rss":
		// RSS fetcher returns []map[string]any with title, link, published, description.
		items, ok := data.([]map[string]any)
		if !ok {
			return nil
		}
		result := make([]upal.SourceItem, 0, len(items))
		for _, item := range items {
			si := upal.SourceItem{
				Title:   stringVal(item, "title"),
				URL:     stringVal(item, "link"),
				Content: stringVal(item, "description"),
			}
			result = append(result, si)
		}
		return result

	case "http":
		// HTTP fetcher returns map[string]any with status, body.
		m, ok := data.(map[string]any)
		if !ok {
			return nil
		}
		body := stringVal(m, "body")
		if body == "" {
			return nil
		}
		return []upal.SourceItem{{Content: body}}

	case "scrape":
		// Scrape fetcher returns []string.
		items, ok := data.([]string)
		if !ok {
			return nil
		}
		result := make([]upal.SourceItem, 0, len(items))
		for _, item := range items {
			result = append(result, upal.SourceItem{Content: item})
		}
		return result
	}

	return nil
}

// stringVal extracts a string value from a map, returning "" if not found or not a string.
func stringVal(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// runAnalysis calls the LLM to analyze collected source data and records
// the analysis result. Errors are logged but do not block session progression.
func (c *ContentCollector) runAnalysis(ctx context.Context, pipeline *upal.Pipeline, session *upal.ContentSession) {
	fetches, err := c.contentSvc.ListSourceFetches(ctx, session.ID)
	if err != nil {
		log.Printf("content_collector: failed to list source fetches for analysis: %v", err)
		return
	}

	llm, modelName, err := c.resolver.Resolve(pipeline.Model)
	if err != nil {
		log.Printf("content_collector: failed to resolve model %q: %v", pipeline.Model, err)
		return
	}

	systemPrompt, userPrompt := buildAnalysisPrompt(pipeline, fetches)

	req := &adkmodel.LLMRequest{
		Model: modelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(userPrompt, genai.RoleUser),
		},
	}

	analysisCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	var resp *adkmodel.LLMResponse
	for r, err := range llm.GenerateContent(analysisCtx, req, false) {
		if err != nil {
			log.Printf("content_collector: LLM analysis failed: %v", err)
			return
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		log.Printf("content_collector: LLM returned empty analysis response")
		return
	}

	text := llmutil.ExtractText(resp)
	stripped, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		log.Printf("content_collector: failed to parse analysis JSON: %v (raw: %.500s)", err, text)
		return
	}

	var parsed struct {
		Summary         string   `json:"summary"`
		Insights        []string `json:"insights"`
		SuggestedAngles []struct {
			Format   string `json:"format"`
			Headline string `json:"headline"`
		} `json:"suggested_angles"`
		OverallScore int `json:"overall_score"`
	}
	if err := json.Unmarshal([]byte(stripped), &parsed); err != nil {
		log.Printf("content_collector: failed to unmarshal analysis: %v (raw: %.500s)", err, stripped)
		return
	}

	// Count total raw items across all fetches.
	totalItems := 0
	for _, sf := range fetches {
		totalItems += len(sf.RawItems)
	}

	angles := make([]upal.ContentAngle, 0, len(parsed.SuggestedAngles))
	for i, a := range parsed.SuggestedAngles {
		angles = append(angles, upal.ContentAngle{
			ID:       fmt.Sprintf("angle-%d", i+1),
			Format:   a.Format,
			Headline: a.Headline,
			Selected: true, // default: all angles selected
		})
	}

	analysis := &upal.LLMAnalysis{
		SessionID:       session.ID,
		RawItemCount:    totalItems,
		FilteredCount:   totalItems, // no filtering applied yet
		Summary:         parsed.Summary,
		Insights:        parsed.Insights,
		SuggestedAngles: angles,
		OverallScore:    parsed.OverallScore,
	}

	if err := c.contentSvc.RecordAnalysis(ctx, analysis); err != nil {
		log.Printf("content_collector: failed to record analysis: %v", err)
	}
}

// buildAnalysisPrompt constructs system and user prompts for the LLM analysis step.
func buildAnalysisPrompt(pipeline *upal.Pipeline, fetches []*upal.SourceFetch) (systemPrompt, userPrompt string) {
	systemPrompt = `You are a content analyst. Analyze the collected data and return a JSON object with these fields:
- summary: 2-3 sentence overview of the collected content
- insights: array of up to 5 key findings as strings
- suggested_angles: array of objects with "format" (one of: blog, shorts, newsletter, longform) and "headline" (short title) fields
- overall_score: 0-100 relevance score based on how well the content matches the pipeline context

Only return valid JSON, no markdown fences, no commentary.`

	var b strings.Builder
	b.WriteString("## Pipeline Context\n")
	if pipeline.Context != nil {
		ctx := pipeline.Context
		if ctx.Purpose != "" {
			fmt.Fprintf(&b, "Purpose: %s\n", ctx.Purpose)
		}
		if ctx.TargetAudience != "" {
			fmt.Fprintf(&b, "Target audience: %s\n", ctx.TargetAudience)
		}
		if ctx.ToneStyle != "" {
			fmt.Fprintf(&b, "Tone/style: %s\n", ctx.ToneStyle)
		}
		if len(ctx.FocusKeywords) > 0 {
			fmt.Fprintf(&b, "Focus keywords: %s\n", strings.Join(ctx.FocusKeywords, ", "))
		}
		if len(ctx.ExcludeKeywords) > 0 {
			fmt.Fprintf(&b, "Exclude keywords: %s\n", strings.Join(ctx.ExcludeKeywords, ", "))
		}
		if ctx.ContentGoals != "" {
			fmt.Fprintf(&b, "Content goals: %s\n", ctx.ContentGoals)
		}
	} else {
		fmt.Fprintf(&b, "Pipeline: %s\n", pipeline.Name)
		if pipeline.Description != "" {
			fmt.Fprintf(&b, "Description: %s\n", pipeline.Description)
		}
	}

	b.WriteString("\n## Collected Items\n\n")
	itemNum := 0
	for _, sf := range fetches {
		if sf.Error != nil {
			continue
		}
		for _, item := range sf.RawItems {
			itemNum++
			fmt.Fprintf(&b, "### Item %d", itemNum)
			if sf.ToolName != "" {
				fmt.Fprintf(&b, " [source: %s]", sf.ToolName)
			}
			b.WriteString("\n")
			if item.Title != "" {
				fmt.Fprintf(&b, "Title: %s\n", item.Title)
			}
			if item.URL != "" {
				fmt.Fprintf(&b, "URL: %s\n", item.URL)
			}
			if item.Content != "" {
				content := item.Content
				if len(content) > 500 {
					content = content[:500] + "..."
				}
				fmt.Fprintf(&b, "Content: %s\n", content)
			}
			b.WriteString("\n")
		}
	}

	if itemNum == 0 {
		b.WriteString("(No items collected)\n")
	}

	userPrompt = b.String()
	return
}

// ProduceWorkflows executes the selected workflows with collected content as input.
// This is designed to run in a background goroutine after session approval.
func (c *ContentCollector) ProduceWorkflows(ctx context.Context, sessionID string, workflowNames []string) {
	// Get session detail for analysis and sources.
	detail, err := c.contentSvc.GetSessionDetail(ctx, sessionID)
	if err != nil {
		log.Printf("content_collector: failed to get session detail for production: %v", err)
		if statusErr := c.contentSvc.UpdateSessionStatus(ctx, sessionID, upal.SessionError); statusErr != nil {
			log.Printf("content_collector: failed to update session status to error: %v", statusErr)
		}
		return
	}

	// Initialize workflow results with pending status.
	results := make([]upal.WorkflowResult, len(workflowNames))
	for i, name := range workflowNames {
		results[i] = upal.WorkflowResult{
			WorkflowName: name,
			Status:       "pending",
		}
	}
	c.contentSvc.SetWorkflowResults(ctx, sessionID, results)

	// Transition session to producing.
	if err := c.contentSvc.UpdateSessionStatus(ctx, sessionID, upal.SessionProducing); err != nil {
		log.Printf("content_collector: failed to transition to producing: %v", err)
	}

	// Execute each workflow sequentially.
	for i, name := range workflowNames {
		// Update status to running.
		results[i].Status = "running"
		c.contentSvc.SetWorkflowResults(ctx, sessionID, results)

		// Look up workflow definition.
		wf, err := c.workflowRepo.Get(ctx, name)
		if err != nil {
			log.Printf("content_collector: workflow %q not found: %v", name, err)
			results[i].Status = "failed"
			c.contentSvc.SetWorkflowResults(ctx, sessionID, results)
			continue
		}

		// Build inputs mapped to actual input node IDs.
		inputs := buildProductionInputs(detail, wf)

		// Run the workflow.
		eventCh, resultCh, err := c.workflowSvc.Run(ctx, wf, inputs)
		if err != nil {
			log.Printf("content_collector: failed to run workflow %q: %v", name, err)
			results[i].Status = "failed"
			c.contentSvc.SetWorkflowResults(ctx, sessionID, results)
			continue
		}

		// Drain event channel, capturing any errors.
		var runErr string
		for evt := range eventCh {
			if evt.Type == "error" {
				if errMsg, ok := evt.Payload["error"].(string); ok {
					runErr = errMsg
				}
			}
		}

		if runErr != "" {
			log.Printf("content_collector: workflow %q execution error: %s", name, runErr)
			results[i].Status = "failed"
			c.contentSvc.SetWorkflowResults(ctx, sessionID, results)
			continue
		}

		// Wait for result.
		runResult, ok := <-resultCh
		if !ok {
			log.Printf("content_collector: workflow %q result channel closed unexpectedly", name)
			results[i].Status = "failed"
			c.contentSvc.SetWorkflowResults(ctx, sessionID, results)
			continue
		}

		now := time.Now()
		results[i].Status = "success"
		results[i].RunID = runResult.SessionID
		results[i].CompletedAt = &now
		c.contentSvc.SetWorkflowResults(ctx, sessionID, results)
	}

	// Determine final status: if any succeeded, move to approved (awaiting publish);
	// if all failed, move to error.
	anySuccess := false
	for _, r := range results {
		if r.Status == "success" {
			anySuccess = true
			break
		}
	}
	finalStatus := upal.SessionError
	if anySuccess {
		finalStatus = upal.SessionApproved
	}
	if err := c.contentSvc.UpdateSessionStatus(ctx, sessionID, finalStatus); err != nil {
		log.Printf("content_collector: failed to transition after produce: %v", err)
	}
}

// buildProductionInputs creates the input map for production workflows.
// Keys are mapped to the workflow's actual input node IDs so the DAG
// runner can inject them into the correct nodes.
func buildProductionInputs(detail *upal.ContentSessionDetail, wf *upal.WorkflowDefinition) map[string]any {
	// Build a combined brief from analysis + sources.
	var sb strings.Builder

	if detail.Analysis != nil {
		fmt.Fprintf(&sb, "## Summary\n%s\n\n", detail.Analysis.Summary)

		if len(detail.Analysis.Insights) > 0 {
			sb.WriteString("## Key Insights\n")
			for _, insight := range detail.Analysis.Insights {
				fmt.Fprintf(&sb, "- %s\n", insight)
			}
			sb.WriteString("\n")
		}
		if len(detail.Analysis.SuggestedAngles) > 0 {
			sb.WriteString("## Suggested Angles\n")
			for _, angle := range detail.Analysis.SuggestedAngles {
				fmt.Fprintf(&sb, "- [%s] %s\n", angle.Format, angle.Headline)
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("## Collected Sources\n\n")
	for _, sf := range detail.Sources {
		if sf.Error != nil {
			continue
		}
		for _, item := range sf.RawItems {
			if item.Title != "" {
				fmt.Fprintf(&sb, "### %s\n", item.Title)
			}
			if item.URL != "" {
				fmt.Fprintf(&sb, "URL: %s\n", item.URL)
			}
			if item.Content != "" {
				content := item.Content
				if len(content) > 500 {
					content = content[:500] + "..."
				}
				fmt.Fprintf(&sb, "%s\n", content)
			}
			sb.WriteString("\n---\n\n")
		}
	}

	brief := sb.String()

	// Map the combined brief to every input node in the workflow.
	inputs := make(map[string]any)
	for _, node := range wf.Nodes {
		if node.Type == "input" {
			inputs[node.ID] = brief
		}
	}

	return inputs
}

// mappedSource pairs a CollectSource with its original index in the pipeline sources.
type mappedSource struct {
	collectSource upal.CollectSource
	pipelineIndex int
}

// mapPipelineSources converts pipeline sources to collect sources using the
// documented mapping rules. Sources with unsupported types are skipped with
// a log warning.
func mapPipelineSources(sources []upal.PipelineSource, isTest bool, limit int) []mappedSource {
	var result []mappedSource

	for i, ps := range sources {
		var cs upal.CollectSource
		cs.ID = ps.ID
		if cs.ID == "" {
			cs.ID = fmt.Sprintf("source-%d", i)
		}

		switch ps.Type {
		case "rss":
			cs.Type = "rss"
			cs.URL = ps.URL
			cs.Limit = ps.Limit

		case "hn":
			cs.Type = "rss"
			url := "https://hnrss.org/newest"
			if ps.MinScore > 0 {
				url = fmt.Sprintf("%s?points=%d", url, ps.MinScore)
			}
			cs.URL = url
			cs.Limit = ps.Limit

		case "reddit":
			cs.Type = "rss"
			subreddit := ps.Subreddit
			if subreddit == "" {
				subreddit = "all"
			}
			cs.URL = fmt.Sprintf("https://www.reddit.com/r/%s/hot/.rss", subreddit)
			cs.Limit = ps.Limit

		case "http":
			cs.Type = "http"
			cs.URL = ps.URL

		case "google_trends", "twitter":
			log.Printf("content_collector: skipping unsupported source type %q (source %s)", ps.Type, ps.ID)
			continue

		default:
			log.Printf("content_collector: skipping unknown source type %q (source %s)", ps.Type, ps.ID)
			continue
		}

		if isTest && limit > 0 {
			cs.Limit = limit
		}

		result = append(result, mappedSource{
			collectSource: cs,
			pipelineIndex: i,
		})
	}

	return result
}
