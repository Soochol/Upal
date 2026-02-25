package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/soochol/upal/internal/llmutil"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/skills"
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
	contentSvc    *ContentSessionService
	collectExec   *CollectStageExecutor
	workflowSvc   *WorkflowService
	workflowRepo  repository.WorkflowRepository
	pipelineRepo  repository.PipelineRepository
	resolver      ports.LLMResolver
	generator     ports.WorkflowGenerator
	skills        skills.Provider
	runHistorySvc ports.RunHistoryPort
}

// NewContentCollector creates a ContentCollector with all required dependencies.
func NewContentCollector(
	contentSvc *ContentSessionService,
	collectExec *CollectStageExecutor,
	workflowSvc *WorkflowService,
	workflowRepo repository.WorkflowRepository,
	pipelineRepo repository.PipelineRepository,
	resolver ports.LLMResolver,
	skills skills.Provider,
	runHistorySvc ports.RunHistoryPort,
) *ContentCollector {
	// Register fetchers that need LLM deps here (not in NewCollectStageExecutor).
	collectExec.RegisterFetcher(NewResearchFetcher(resolver, skills))

	return &ContentCollector{
		contentSvc:    contentSvc,
		collectExec:   collectExec,
		workflowSvc:   workflowSvc,
		workflowRepo:  workflowRepo,
		pipelineRepo:  pipelineRepo,
		resolver:      resolver,
		skills:        skills,
		runHistorySvc: runHistorySvc,
	}
}

// SetGenerator injects the workflow generator (called after both collector and generator are created).
func (c *ContentCollector) SetGenerator(g ports.WorkflowGenerator) {
	c.generator = g
}

// CollectPipeline creates a session for the given pipeline and launches background
// collection + analysis. Designed for scheduled/automated invocations.
func (c *ContentCollector) CollectPipeline(ctx context.Context, pipelineID string) error {
	pipeline, err := c.pipelineRepo.Get(ctx, pipelineID)
	if err != nil {
		return fmt.Errorf("pipeline %s: %w", pipelineID, err)
	}
	sess := &upal.ContentSession{
		PipelineID:  pipelineID,
		TriggerType: "scheduled",
		IsTemplate:  false,
		Sources:     pipeline.Sources,
		Schedule:    pipeline.Schedule,
		Model:       pipeline.Model,
		Workflows:   pipeline.Workflows,
		Context:     pipeline.Context,
	}
	if err := c.contentSvc.CreateSession(ctx, sess); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	go c.CollectAndAnalyze(context.Background(), sess, false, 0)
	return nil
}

// CollectFromTemplate creates a child instance from a session template and
// launches background collection + analysis.
func (c *ContentCollector) CollectFromTemplate(ctx context.Context, templateID string) error {
	template, err := c.contentSvc.GetSession(ctx, templateID)
	if err != nil {
		return fmt.Errorf("template %s: %w", templateID, err)
	}
	if !template.IsTemplate {
		return fmt.Errorf("session %s is not a template", templateID)
	}

	instanceName := template.Name
	if instanceName != "" {
		instanceName += " — " + time.Now().Format("01/02 15:04")
	}
	sess := &upal.ContentSession{
		PipelineID:      template.PipelineID,
		Name:            instanceName,
		TriggerType:     "schedule",
		IsTemplate:      false,
		ParentSessionID: template.ID,
		Sources:         template.Sources,
		Model:           template.Model,
		Workflows:       template.Workflows,
		Context:         template.Context,
	}
	if err := c.contentSvc.CreateSession(ctx, sess); err != nil {
		return fmt.Errorf("create instance: %w", err)
	}
	go c.CollectAndAnalyze(context.Background(), sess, false, 0)
	return nil
}

// CollectAndAnalyze fetches content from session sources, records the results,
// runs LLM analysis on the collected data, and transitions the session to
// pending_review. This is designed to run in a background goroutine.
//
// If isTest is true, each source's item limit is overridden with the provided
// limit value for faster iteration.
func (c *ContentCollector) CollectAndAnalyze(ctx context.Context, session *upal.ContentSession, isTest bool, limit int) {
	// Map session sources to collect sources.
	sources := mapPipelineSources(session.Sources, session.Model, isTest, limit)

	if len(sources) == 0 {
		log.Printf("content_collector: no fetchable sources for session %s", session.ID)
		if err := c.contentSvc.UpdateSessionStatus(ctx, session.ID, upal.SessionPendingReview); err != nil {
			log.Printf("content_collector: failed to update session status: %v", err)
		}
		return
	}

	// Fetch each source and record results.
	totalItems := 0
	for _, mapped := range sources {
		pipelineSrc := session.Sources[mapped.pipelineIndex]
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
		c.runAnalysis(ctx, session)
	}

	// Transition to pending_review regardless of analysis outcome.
	if err := c.contentSvc.UpdateSessionStatus(ctx, session.ID, upal.SessionPendingReview); err != nil {
		log.Printf("content_collector: failed to transition to pending_review: %v", err)
	}
}

// RetryAnalysis re-runs LLM analysis for a session that is stuck in "analyzing"
// state (e.g. due to server restart during analysis). It uses the session's own
// settings, runs analysis, and always transitions the session to pending_review.
func (c *ContentCollector) RetryAnalysis(ctx context.Context, sessionID string) error {
	session, err := c.contentSvc.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Reset to analyzing so the frontend sees immediate feedback.
	if err := c.contentSvc.UpdateSessionStatus(ctx, session.ID, upal.SessionAnalyzing); err != nil {
		return fmt.Errorf("failed to transition to analyzing: %w", err)
	}

	go func() {
		bgCtx := context.Background()
		c.runAnalysis(bgCtx, session)
		if err := c.contentSvc.UpdateSessionStatus(bgCtx, session.ID, upal.SessionPendingReview); err != nil {
			log.Printf("content_collector: retry-analyze failed to transition to pending_review: %v", err)
		}
	}()
	return nil
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

	case "social":
		// Social fetcher returns []map[string]any with title, url, content, fetched_from.
		items, ok := data.([]map[string]any)
		if !ok {
			return nil
		}
		result := make([]upal.SourceItem, 0, len(items))
		for _, item := range items {
			result = append(result, upal.SourceItem{
				Title:       stringVal(item, "title"),
				URL:         stringVal(item, "url"),
				Content:     stringVal(item, "content"),
				FetchedFrom: stringVal(item, "fetched_from"),
			})
		}
		return result

	case "research":
		// Research fetcher returns []map[string]any with title, url, summary.
		items, ok := data.([]map[string]any)
		if !ok {
			return nil
		}
		result := make([]upal.SourceItem, 0, len(items))
		for _, item := range items {
			result = append(result, upal.SourceItem{
				Title:   stringVal(item, "title"),
				URL:     stringVal(item, "url"),
				Content: stringVal(item, "summary"),
			})
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
func (c *ContentCollector) runAnalysis(ctx context.Context, session *upal.ContentSession) {
	fetches, err := c.contentSvc.ListSourceFetches(ctx, session.ID)
	if err != nil {
		log.Printf("content_collector: failed to list source fetches for analysis: %v", err)
		return
	}

	// Fetch all available workflows for smart matching.
	allWorkflows, err := c.workflowRepo.List(ctx)
	if err != nil {
		log.Printf("content_collector: failed to list workflows for analysis: %v", err)
		allWorkflows = nil // non-fatal
	}

	llm, modelName, err := c.resolver.Resolve(session.Model)
	if err != nil {
		log.Printf("content_collector: failed to resolve model %q: %v", session.Model, err)
		return
	}

	systemPrompt, userPrompt := buildAnalysisPrompt(c.skills.GetPrompt("content-analyze"), session.Context, session.Workflows, fetches, allWorkflows)

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
			Format       string `json:"format"`
			Headline     string `json:"headline"`
			WorkflowName string `json:"workflow_name"`
			Rationale    string `json:"rationale"`
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

	// Build a set of valid workflow names for validation.
	validWorkflows := make(map[string]bool)
	for _, wf := range allWorkflows {
		validWorkflows[wf.Name] = true
	}

	angles := make([]upal.ContentAngle, 0, len(parsed.SuggestedAngles))
	for i, a := range parsed.SuggestedAngles {
		workflowName := a.WorkflowName
		matchType := "none"

		// Validate that the LLM didn't hallucinate a workflow name.
		if workflowName != "" && validWorkflows[workflowName] {
			matchType = "matched"
		} else {
			workflowName = ""
		}

		angles = append(angles, upal.ContentAngle{
			ID:           fmt.Sprintf("angle-%d", i+1),
			Format:       a.Format,
			Headline:     a.Headline,
			Rationale:    a.Rationale,
			Selected:     true,
			WorkflowName: workflowName,
			MatchType:    matchType,
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
func buildAnalysisPrompt(systemPromptBase string, pipelineContext *upal.PipelineContext, sessionWorkflows []upal.PipelineWorkflow, fetches []*upal.SourceFetch, workflows []*upal.WorkflowDefinition) (systemPrompt, userPrompt string) {
	systemPrompt = systemPromptBase

	var b strings.Builder
	b.WriteString("## Pipeline Context\n")
	if pipelineContext != nil {
		if pipelineContext.Purpose != "" {
			fmt.Fprintf(&b, "Purpose: %s\n", pipelineContext.Purpose)
		}
		if pipelineContext.TargetAudience != "" {
			fmt.Fprintf(&b, "Target audience: %s\n", pipelineContext.TargetAudience)
		}
		if pipelineContext.ToneStyle != "" {
			fmt.Fprintf(&b, "Tone/style: %s\n", pipelineContext.ToneStyle)
		}
		if len(pipelineContext.FocusKeywords) > 0 {
			fmt.Fprintf(&b, "Focus keywords: %s\n", strings.Join(pipelineContext.FocusKeywords, ", "))
		}
		if len(pipelineContext.ExcludeKeywords) > 0 {
			fmt.Fprintf(&b, "Exclude keywords: %s\n", strings.Join(pipelineContext.ExcludeKeywords, ", "))
		}
		if pipelineContext.ContentGoals != "" {
			fmt.Fprintf(&b, "Content goals: %s\n", pipelineContext.ContentGoals)
		}
	} else {
		b.WriteString("(No editorial context provided)\n")
	}

	if len(sessionWorkflows) > 0 {
		b.WriteString("\n## Pipeline's Preferred Workflows\n")
		for _, pw := range sessionWorkflows {
			fmt.Fprintf(&b, "- %s", pw.WorkflowName)
			if pw.Label != "" {
				fmt.Fprintf(&b, " (%s)", pw.Label)
			}
			b.WriteString("\n")
		}
	}

	if len(workflows) > 0 {
		b.WriteString("\n## Available Workflows\n")
		for _, wf := range workflows {
			fmt.Fprintf(&b, "- %q", wf.Name)
			if wf.Description != "" {
				fmt.Fprintf(&b, ": %s", wf.Description)
			}
			b.WriteString("\n")
			for _, n := range wf.Nodes {
				label, _ := n.Config["label"].(string)
				desc, _ := n.Config["description"].(string)
				if label != "" || desc != "" {
					fmt.Fprintf(&b, "  [%s] %s", n.Type, label)
					if desc != "" {
						fmt.Fprintf(&b, " — %s", desc)
					}
					b.WriteString("\n")
				}
			}
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

// WorkflowRequest represents a workflow to execute with an optional channel assignment.
type WorkflowRequest struct {
	Name      string
	ChannelID string
}

// ProduceWorkflows executes the selected workflows with collected content as input.
// This is designed to run in a background goroutine after session approval.
func (c *ContentCollector) ProduceWorkflows(ctx context.Context, sessionID string, requests []WorkflowRequest) {
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
	results := make([]upal.WorkflowResult, len(requests))
	for i, req := range requests {
		results[i] = upal.WorkflowResult{
			WorkflowName: req.Name,
			Status:       upal.WFResultPending,
			ChannelID:    req.ChannelID,
		}
	}
	c.contentSvc.SetWorkflowResults(ctx, sessionID, results)

	// Transition session to producing.
	if err := c.contentSvc.UpdateSessionStatus(ctx, sessionID, upal.SessionProducing); err != nil {
		log.Printf("content_collector: failed to transition to producing: %v", err)
	}

	// Execute workflows in parallel.
	var mu sync.Mutex
	updateResult := func(i int, fn func(*upal.WorkflowResult)) {
		mu.Lock()
		fn(&results[i])
		c.contentSvc.SetWorkflowResults(ctx, sessionID, results)
		mu.Unlock()
	}

	g, gCtx := errgroup.WithContext(ctx)
	for i, req := range requests {
		g.Go(func() error {
			updateResult(i, func(r *upal.WorkflowResult) { r.Status = upal.WFResultRunning })

			// Look up workflow definition.
			wf, err := c.workflowRepo.Get(gCtx, req.Name)
			if err != nil {
				log.Printf("content_collector: workflow %q not found: %v", req.Name, err)
				updateResult(i, func(r *upal.WorkflowResult) {
					r.Status = upal.WFResultFailed
					r.ErrorMessage = fmt.Sprintf("workflow not found: %v", err)
				})
				return nil
			}

			// Build inputs mapped to actual input node IDs.
			inputs := buildProductionInputs(detail, wf)

			// Create a RunRecord if history service is available.
			var runRecordID string
			if c.runHistorySvc != nil {
				rec, startErr := c.runHistorySvc.StartRun(gCtx, req.Name, "pipeline", sessionID, inputs, wf)
				if startErr != nil {
					log.Printf("content_collector: failed to start run record for %q: %v", req.Name, startErr)
				} else {
					runRecordID = rec.ID
				}
			}

			// Run the workflow.
			eventCh, resultCh, err := c.workflowSvc.Run(gCtx, wf, inputs)
			if err != nil {
				errMsg := fmt.Sprintf("failed to start workflow: %v", err)
				log.Printf("content_collector: %s %q", errMsg, req.Name)
				if c.runHistorySvc != nil && runRecordID != "" {
					c.runHistorySvc.FailRun(gCtx, runRecordID, errMsg)
				}
				updateResult(i, func(r *upal.WorkflowResult) {
					r.Status = upal.WFResultFailed
					r.ErrorMessage = errMsg
					r.RunID = runRecordID
				})
				return nil
			}

			// Drain event channel, tracking node runs and capturing errors.
			var runErr string
			var failedNodeID string
			for evt := range eventCh {
				if c.runHistorySvc != nil && runRecordID != "" {
					trackNodeRunFromEvent(gCtx, c.runHistorySvc, runRecordID, evt)
				}
				if evt.Type == upal.EventError {
					if errMsg, ok := evt.Payload["error"].(string); ok {
						runErr = errMsg
					}
					if evt.NodeID != "" {
						failedNodeID = evt.NodeID
					}
				}
			}

			if runErr != "" {
				log.Printf("content_collector: workflow %q execution error: %s", req.Name, runErr)
				if c.runHistorySvc != nil && runRecordID != "" {
					c.runHistorySvc.FailRun(gCtx, runRecordID, runErr)
					// If no failed node from events, check the RunRecord.
					if failedNodeID == "" {
						if rec, err := c.runHistorySvc.GetRun(gCtx, runRecordID); err == nil && rec != nil {
							for _, nr := range rec.NodeRuns {
								if nr.Status == upal.NodeRunError {
									failedNodeID = nr.NodeID
									break
								}
							}
						}
					}
				}
				updateResult(i, func(r *upal.WorkflowResult) {
					r.Status = upal.WFResultFailed
					r.RunID = runRecordID
					r.ErrorMessage = runErr
					r.FailedNodeID = failedNodeID
				})
				return nil
			}

			// Wait for result.
			runResult, ok := <-resultCh
			if !ok {
				errMsg := "result channel closed unexpectedly"
				log.Printf("content_collector: workflow %q %s", req.Name, errMsg)
				if c.runHistorySvc != nil && runRecordID != "" {
					c.runHistorySvc.FailRun(gCtx, runRecordID, errMsg)
				}
				updateResult(i, func(r *upal.WorkflowResult) {
					r.Status = upal.WFResultFailed
					r.RunID = runRecordID
					r.ErrorMessage = errMsg
				})
				return nil
			}

			// Mark run as completed with outputs.
			if c.runHistorySvc != nil && runRecordID != "" {
				c.runHistorySvc.CompleteRun(gCtx, runRecordID, runResult.State)
			}

			now := time.Now()
			updateResult(i, func(r *upal.WorkflowResult) {
				r.Status = upal.WFResultSuccess
				r.RunID = runRecordID
				r.CompletedAt = &now
			})
			return nil
		})
	}
	_ = g.Wait()

	// Determine final status: if any succeeded, move to approved (awaiting publish);
	// if all failed, move to error.
	anySuccess := false
	for _, r := range results {
		if r.Status == upal.WFResultSuccess {
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

// GenerateWorkflowForAngle creates a new workflow for an unmatched content angle
// using the Generator, saves it to the workflow repo, and updates the analysis.
func (c *ContentCollector) GenerateWorkflowForAngle(ctx context.Context, sessionID, angleID string) (*upal.ContentAngle, error) {
	if c.generator == nil {
		return nil, fmt.Errorf("workflow generator not available")
	}

	// Get the analysis and find the angle.
	analysis, err := c.contentSvc.GetAnalysis(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get analysis: %w", err)
	}
	if analysis == nil {
		return nil, fmt.Errorf("no analysis found for session %s", sessionID)
	}

	var targetIdx int = -1
	for i := range analysis.SuggestedAngles {
		if analysis.SuggestedAngles[i].ID == angleID {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return nil, fmt.Errorf("angle %q not found in session %s", angleID, sessionID)
	}

	angle := &analysis.SuggestedAngles[targetIdx]

	// Build a description enriched with session context.
	var desc strings.Builder
	fmt.Fprintf(&desc, "Create a %s content production workflow that takes collected source material as input and produces a polished %s.\nTarget headline: %q\n",
		angle.Format, angle.Format, angle.Headline)

	// Inject editorial brief from the session's own context.
	session, err := c.contentSvc.GetSession(ctx, sessionID)
	if err == nil && session != nil && session.Context != nil {
		pctx := session.Context
		if pctx.Purpose != "" {
			fmt.Fprintf(&desc, "Purpose: %s\n", pctx.Purpose)
		}
		if pctx.TargetAudience != "" {
			fmt.Fprintf(&desc, "Target audience: %s\n", pctx.TargetAudience)
		}
		if pctx.ToneStyle != "" {
			fmt.Fprintf(&desc, "Tone/style: %s\n", pctx.ToneStyle)
		}
		if pctx.ContentGoals != "" {
			fmt.Fprintf(&desc, "Content goals: %s\n", pctx.ContentGoals)
		}
		if pctx.Language != "" {
			fmt.Fprintf(&desc, "Output language: %s\n", pctx.Language)
		}
	}

	// Generate the workflow.
	wf, err := c.generator.GenerateWorkflow(ctx, desc.String())
	if err != nil {
		return nil, fmt.Errorf("generate workflow: %w", err)
	}

	// Save the workflow to the repo.
	if err := c.workflowRepo.Create(ctx, wf); err != nil {
		// Name conflict — try with a suffix.
		wf.Name = wf.Name + "-" + upal.GenerateID("")[:6]
		if err2 := c.workflowRepo.Create(ctx, wf); err2 != nil {
			return nil, fmt.Errorf("save generated workflow: %w", err2)
		}
	}

	// Update the angle in the analysis.
	angle.WorkflowName = wf.Name
	angle.MatchType = "generated"

	if err := c.contentSvc.UpdateAnalysisAngles(ctx, sessionID, analysis.SuggestedAngles); err != nil {
		log.Printf("content_collector: failed to update analysis angles: %v", err)
		// Non-fatal: workflow was created, angle just won't persist the link
	}

	return angle, nil
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
func mapPipelineSources(sources []upal.PipelineSource, defaultModel string, isTest bool, limit int) []mappedSource {
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

		case "google_trends":
			cs.Type = "rss"
			geo := ps.Geo
			if geo == "" {
				geo = "US"
			}
			cs.URL = fmt.Sprintf("https://trends.google.com/trending/rss?geo=%s", geo)
			cs.Limit = ps.Limit

		case "social", "twitter":
			cs.Type = "social"
			cs.Keywords = ps.Keywords
			cs.Accounts = ps.Accounts
			cs.Limit = ps.Limit

		case "research":
			cs.Type = "research"
			cs.Topic = ps.Topic
			cs.Depth = ps.Depth
			if cs.Depth == "" {
				cs.Depth = "light"
			}
			cs.Model = defaultModel

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

// trackNodeRunFromEvent records node-level execution status from workflow events.
func trackNodeRunFromEvent(ctx context.Context, svc ports.RunHistoryPort, runID string, ev upal.WorkflowEvent) {
	if ev.NodeID == "" {
		return
	}
	now := time.Now()
	switch ev.Type {
	case upal.EventNodeStarted:
		svc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:    ev.NodeID,
			Status:    upal.NodeRunRunning,
			StartedAt: now,
		})
	case upal.EventNodeCompleted:
		var usage *upal.TokenUsage
		if tokens, ok := ev.Payload["tokens"].(map[string]any); ok {
			usage = &upal.TokenUsage{
				PromptTokens:     int32(toIntVal(tokens["prompt_token_count"])),
				CompletionTokens: int32(toIntVal(tokens["candidates_token_count"])),
				TotalTokens:      int32(toIntVal(tokens["total_token_count"])),
			}
		}
		svc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:      ev.NodeID,
			Status:      upal.NodeRunCompleted,
			StartedAt:   now,
			CompletedAt: &now,
			Usage:       usage,
		})
	case upal.EventError:
		svc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:      ev.NodeID,
			Status:      upal.NodeRunError,
			StartedAt:   now,
			CompletedAt: &now,
		})
	}
}

func toIntVal(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int32:
		return int(n)
	default:
		return 0
	}
}
