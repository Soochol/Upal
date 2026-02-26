package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/llmutil"
	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/tools"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

var _ ports.WorkflowExecutor = (*WorkflowService)(nil)

// WorkflowService orchestrates workflow execution: DAG agent creation,
// ADK runner lifecycle, session management, and event classification.
type WorkflowService struct {
	repo           repository.WorkflowRepository
	llms           map[string]adkmodel.LLM
	llmResolver    ports.LLMResolver
	sessionService session.Service
	toolReg        *tools.Registry
	nodeRegistry   *agents.NodeRegistry
	buildDeps      agents.BuildDeps
}

func NewWorkflowService(
	repo repository.WorkflowRepository,
	llms map[string]adkmodel.LLM,
	sessionService session.Service,
	toolReg *tools.Registry,
	nodeRegistry *agents.NodeRegistry,
	outputDir string,
	htmlLayoutPrompt string,
	resolver ports.LLMResolver,
) *WorkflowService {
	return &WorkflowService{
		repo:           repo,
		llms:           llms,
		llmResolver:    resolver,
		sessionService: sessionService,
		toolReg:        toolReg,
		nodeRegistry:   nodeRegistry,
		buildDeps:      agents.BuildDeps{LLMs: llms, LLMResolver: resolver, ToolReg: toolReg, OutputDir: outputDir, HTMLLayoutPrompt: htmlLayoutPrompt},
	}
}

func (s *WorkflowService) Lookup(ctx context.Context, name string) (*upal.WorkflowDefinition, error) {
	return s.repo.Get(ctx, name)
}

func (s *WorkflowService) Validate(wf *upal.WorkflowDefinition) error {
	for _, n := range wf.Nodes {
		if n.Type != upal.NodeTypeAgent {
			continue
		}
		modelID, _ := n.Config["model"].(string)
		if modelID == "" {
			label, _ := n.Config["label"].(string)
			if label == "" {
				label = n.ID
			}
			return fmt.Errorf("node %q has no model selected — please choose a model before running", label)
		}
		if _, _, err := s.llmResolver.Resolve(modelID); err != nil {
			return fmt.Errorf("node %q: %w", n.ID, err)
		}
	}
	return nil
}

func (s *WorkflowService) Run(ctx context.Context, wf *upal.WorkflowDefinition, inputs map[string]any) (<-chan upal.WorkflowEvent, <-chan upal.RunResult, error) {
	dagAgent, err := agents.NewDAGAgent(wf, s.nodeRegistry, s.buildDeps)
	if err != nil {
		return nil, nil, fmt.Errorf("build DAG: %w", err)
	}

	adkRunner, err := runner.New(runner.Config{
		AppName:        wf.Name,
		Agent:          dagAgent,
		SessionService: s.sessionService,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create runner: %w", err)
	}

	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())
	userID := upal.UserIDFromContext(ctx)

	inputState := make(map[string]any)
	for k, v := range inputs {
		inputState["__user_input__"+k] = v
	}

	_, err = s.sessionService.Create(ctx, &session.CreateRequest{
		AppName:   wf.Name,
		UserID:    userID,
		SessionID: sessionID,
		State:     inputState,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	eventCh := make(chan upal.WorkflowEvent, 64)
	resultCh := make(chan upal.RunResult, 1)

	go func() {
		defer close(eventCh)
		defer close(resultCh)

		// done is closed when this goroutine exits, guarding eventCh sends
		// from log callbacks that may fire during teardown.
		done := make(chan struct{})
		defer close(done)

		nodeLogFn := agents.NodeLogFunc(func(nodeID, msg string) {
			slog.Info("model-log", "node", nodeID, "msg", msg)
			select {
			case <-done:
				return
			default:
			}
			defer func() { recover() }()
			eventCh <- upal.WorkflowEvent{
				Type:   "log",
				NodeID: nodeID,
				Payload: map[string]any{"node_id": nodeID, "message": msg},
			}
		})
		logCtx := agents.WithNodeLogFunc(ctx, nodeLogFn)

		userContent := genai.NewContentFromText("run", genai.RoleUser)
		for event, err := range adkRunner.Run(logCtx, userID, sessionID, userContent, agent.RunConfig{}) {
			if err != nil {
				eventCh <- upal.WorkflowEvent{
					Type:    upal.EventError,
					Payload: map[string]any{"error": err.Error()},
				}
				return
			}
			if event == nil {
				continue
			}
			wfEvent := classifyEvent(event)
			eventCh <- wfEvent
		}

		finalState := make(map[string]any)
		getResp, err := s.sessionService.Get(ctx, &session.GetRequest{
			AppName:   wf.Name,
			UserID:    userID,
			SessionID: sessionID,
		})
		if err == nil {
			for k, v := range getResp.Session.State().All() {
				if !strings.HasPrefix(k, "__") {
					finalState[k] = v
				}
			}
		}

		// Collect output node results under __output__ for deterministic frontend access.
		outputs := make(map[string]any)
		for _, n := range wf.Nodes {
			if n.Type == upal.NodeTypeOutput {
				if v, ok := finalState[n.ID]; ok {
					outputs[n.ID] = v
				}
			}
		}
		if len(outputs) > 0 {
			finalState["__output__"] = outputs
		}

		resultCh <- upal.RunResult{
			SessionID: sessionID,
			State:     finalState,
		}
	}()

	return eventCh, resultCh, nil
}

func classifyEvent(event *session.Event) upal.WorkflowEvent {
	nodeID := event.Author
	content := event.LLMResponse.Content

	if status, ok := event.Actions.StateDelta["__status__"].(string); ok {
		switch status {
		case "skipped":
			return upal.WorkflowEvent{Type: upal.EventNodeSkipped, NodeID: nodeID, Payload: map[string]any{"node_id": nodeID}}
		case "waiting":
			return upal.WorkflowEvent{Type: upal.EventNodeWaiting, NodeID: nodeID, Payload: map[string]any{"node_id": nodeID}}
		}
	}

	if content == nil || len(content.Parts) == 0 {
		// Flush events with FinishReason but no content parts are completions, not starts.
		if fr := event.LLMResponse.FinishReason; fr != "" && fr != genai.FinishReasonUnspecified {
			flushOutput := ""
			if a, ok := event.Actions.StateDelta[nodeID]; ok && a != nil {
				flushOutput = fmt.Sprintf("%v", a)
			}
			flushPayload := map[string]any{
				"node_id":      nodeID,
				"output":       flushOutput,
				"state_delta":  event.Actions.StateDelta,
				"finish_reason": string(fr),
			}
			if u := event.LLMResponse.UsageMetadata; u != nil {
				flushPayload["tokens"] = map[string]any{
					"input":  u.PromptTokenCount,
					"output": u.CandidatesTokenCount,
					"total":  u.TotalTokenCount,
				}
			}
			return upal.WorkflowEvent{Type: upal.EventNodeCompleted, NodeID: nodeID, Payload: flushPayload}
		}
		return upal.WorkflowEvent{Type: upal.EventNodeStarted, NodeID: nodeID, Payload: map[string]any{"node_id": nodeID}}
	}

	if hasFunctionCalls(content.Parts) {
		return upal.WorkflowEvent{
			Type:   upal.EventToolCall,
			NodeID: nodeID,
			Payload: map[string]any{
				"node_id": nodeID,
				"calls":   extractFunctionCalls(content.Parts),
			},
		}
	}

	if hasFunctionResponses(content.Parts) {
		return upal.WorkflowEvent{
			Type:   upal.EventToolResult,
			NodeID: nodeID,
			Payload: map[string]any{
				"node_id": nodeID,
				"results": extractFunctionResponses(content.Parts),
			},
		}
	}

	outputStr := llmutil.ExtractContent(&event.LLMResponse)
	if a, ok := event.Actions.StateDelta[nodeID]; ok && a != nil {
		outputStr = fmt.Sprintf("%v", a)
	}
	payload := map[string]any{
		"node_id":     nodeID,
		"output":      outputStr,
		"state_delta": event.Actions.StateDelta,
	}

	if u := event.LLMResponse.UsageMetadata; u != nil {
		payload["tokens"] = map[string]any{
			"input":  u.PromptTokenCount,
			"output": u.CandidatesTokenCount,
			"total":  u.TotalTokenCount,
		}
	}

	if fr := event.LLMResponse.FinishReason; fr != "" && fr != genai.FinishReasonUnspecified {
		payload["finish_reason"] = string(fr)
	}

	return upal.WorkflowEvent{
		Type:    upal.EventNodeCompleted,
		NodeID:  nodeID,
		Payload: payload,
	}
}

func hasFunctionCalls(parts []*genai.Part) bool {
	for _, p := range parts {
		if p.FunctionCall != nil {
			return true
		}
	}
	return false
}

func hasFunctionResponses(parts []*genai.Part) bool {
	for _, p := range parts {
		if p.FunctionResponse != nil {
			return true
		}
	}
	return false
}

func extractFunctionCalls(parts []*genai.Part) []map[string]any {
	var calls []map[string]any
	for _, p := range parts {
		if p.FunctionCall != nil {
			calls = append(calls, map[string]any{"name": p.FunctionCall.Name, "args": p.FunctionCall.Args})
		}
	}
	return calls
}

func extractFunctionResponses(parts []*genai.Part) []map[string]any {
	var results []map[string]any
	for _, p := range parts {
		if p.FunctionResponse != nil {
			results = append(results, map[string]any{"name": p.FunctionResponse.Name, "response": p.FunctionResponse.Response})
		}
	}
	return results
}
