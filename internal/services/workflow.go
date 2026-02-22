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

// Compile-time assertion: WorkflowService must satisfy ports.WorkflowExecutor.
var _ ports.WorkflowExecutor = (*WorkflowService)(nil)

// WorkflowService encapsulates workflow execution orchestration:
// DAG agent creation, ADK runner lifecycle, session management,
// and event classification.
type WorkflowService struct {
	repo           repository.WorkflowRepository
	llms           map[string]adkmodel.LLM
	sessionService session.Service
	toolReg        *tools.Registry
	nodeRegistry   *agents.NodeRegistry
	buildDeps      agents.BuildDeps
}

// NewWorkflowService creates a WorkflowService with all required dependencies.
func NewWorkflowService(
	repo repository.WorkflowRepository,
	llms map[string]adkmodel.LLM,
	sessionService session.Service,
	toolReg *tools.Registry,
	nodeRegistry *agents.NodeRegistry,
	outputDir string,
) *WorkflowService {
	return &WorkflowService{
		repo:           repo,
		llms:           llms,
		sessionService: sessionService,
		toolReg:        toolReg,
		nodeRegistry:   nodeRegistry,
		buildDeps:      agents.BuildDeps{LLMs: llms, ToolReg: toolReg, OutputDir: outputDir},
	}
}

// Lookup resolves a workflow by name via the repository.
func (s *WorkflowService) Lookup(ctx context.Context, name string) (*upal.WorkflowDefinition, error) {
	return s.repo.Get(ctx, name)
}

// Validate checks that all agent nodes have a valid model configured.
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
		parts := strings.SplitN(modelID, "/", 2)
		if len(parts) != 2 || parts[1] == "" {
			return fmt.Errorf("node %q has invalid model format %q — expected \"provider/model\"", n.ID, modelID)
		}
		if _, ok := s.llms[parts[0]]; !ok {
			return fmt.Errorf("node %q uses provider %q which is not configured", n.ID, parts[0])
		}
	}
	return nil
}

// Run executes a workflow and streams events through a channel.
// The caller receives WorkflowEvents as they occur, and a RunResult
// when execution completes. The events channel is closed when done.
func (s *WorkflowService) Run(ctx context.Context, wf *upal.WorkflowDefinition, inputs map[string]any) (<-chan upal.WorkflowEvent, <-chan upal.RunResult, error) {
	// 1. Build DAGAgent from workflow.
	dagAgent, err := agents.NewDAGAgent(wf, s.nodeRegistry, s.buildDeps)
	if err != nil {
		return nil, nil, fmt.Errorf("build DAG: %w", err)
	}

	// 2. Create ADK Runner.
	adkRunner, err := runner.New(runner.Config{
		AppName:        wf.Name,
		Agent:          dagAgent,
		SessionService: s.sessionService,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create runner: %w", err)
	}

	// 3. Create session with user inputs.
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())
	userID := "default"

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

	// 4. Stream events through channels.
	eventCh := make(chan upal.WorkflowEvent, 64)
	resultCh := make(chan upal.RunResult, 1)

	go func() {
		defer close(eventCh)
		defer close(resultCh)

		// done is closed when this goroutine exits, signalling log
		// callbacks that eventCh is no longer accepting sends.
		done := make(chan struct{})
		defer close(done)

		// Wire up model-level logging into the event stream.
		// NOTE: select does NOT protect against sending on a closed channel
		// — Go panics during case evaluation. We check done first and use
		// recover() as a safety net for the race between done-check and
		// eventCh closure.
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

		// Collect final session state.
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

		// Expose output node results under a dedicated key so the frontend
		// can locate the final output deterministically (Go map JSON
		// serialization order is random).
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

// classifyEvent inspects an ADK session.Event and returns a WorkflowEvent
// with the appropriate type and normalized payload.
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

	return upal.WorkflowEvent{
		Type:   upal.EventNodeCompleted,
		NodeID: nodeID,
		Payload: map[string]any{
			"node_id":     nodeID,
			"output":      llmutil.ExtractContent(&event.LLMResponse),
			"state_delta": event.Actions.StateDelta,
		},
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
