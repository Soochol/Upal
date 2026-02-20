package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// upalA2AExecutor implements a2asrv.AgentExecutor to expose Upal workflows
// as A2A-callable agents. It delegates workflow execution to WorkflowService.
type upalA2AExecutor struct {
	workflowSvc *services.WorkflowService
}

func (e *upalA2AExecutor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	// 1. Parse the incoming A2A message.
	workflowName, inputs, err := parseA2AMessage(reqCtx.Message)
	if err != nil {
		return writeFailEvent(ctx, reqCtx, queue, err)
	}

	// 2. Look up the workflow via service.
	wf, err := e.workflowSvc.Lookup(ctx, workflowName)
	if err != nil {
		return writeFailEvent(ctx, reqCtx, queue, fmt.Errorf("workflow %q not found", workflowName))
	}

	// 3. Submit task.
	if reqCtx.StoredTask == nil {
		event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateSubmitted, nil)
		if err := queue.Write(ctx, event); err != nil {
			return fmt.Errorf("failed to write submitted: %w", err)
		}
	}

	// 4. Working status.
	workingEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateWorking, nil)
	if err := queue.Write(ctx, workingEvent); err != nil {
		return fmt.Errorf("failed to write working: %w", err)
	}

	// 5. Execute via WorkflowService.
	events, _, runErr := e.workflowSvc.Run(ctx, wf, inputs)
	if runErr != nil {
		return writeFailEvent(ctx, reqCtx, queue, fmt.Errorf("failed to run workflow: %w", runErr))
	}

	// 6. Stream events as A2A artifacts.
	var artifactID a2a.ArtifactID
	for ev := range events {
		if ev.Type == "error" {
			return writeFailEvent(ctx, reqCtx, queue, fmt.Errorf("%v", ev.Payload["error"]))
		}

		text := extractPayloadText(ev)
		if text == "" {
			continue
		}

		var artEvent *a2a.TaskArtifactUpdateEvent
		if artifactID == "" {
			artEvent = a2a.NewArtifactEvent(reqCtx, a2a.TextPart{Text: text})
			artifactID = artEvent.Artifact.ID
		} else {
			artEvent = a2a.NewArtifactUpdateEvent(reqCtx, artifactID, a2a.TextPart{Text: text})
		}

		if err := queue.Write(ctx, artEvent); err != nil {
			return fmt.Errorf("failed to write artifact: %w", err)
		}
	}

	// 7. Completed.
	doneEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCompleted, nil)
	doneEvent.Final = true
	if err := queue.Write(ctx, doneEvent); err != nil {
		return fmt.Errorf("failed to write completed: %w", err)
	}
	return nil
}

func (e *upalA2AExecutor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCanceled, nil)
	event.Final = true
	return queue.Write(ctx, event)
}

// extractPayloadText pulls text from a WorkflowEvent payload.
func extractPayloadText(ev services.WorkflowEvent) string {
	if ev.Type != services.EventNodeCompleted {
		return ""
	}
	if output, ok := ev.Payload["output"].(string); ok {
		return output
	}
	return ""
}

// writeFailEvent sends a TaskStateFailed event with the error message.
func writeFailEvent(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue, err error) error {
	msg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.TextPart{Text: err.Error()})
	event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateFailed, msg)
	event.Final = true
	if writeErr := queue.Write(ctx, event); writeErr != nil {
		return fmt.Errorf("failed to write failure event: %w (original: %v)", writeErr, err)
	}
	return nil
}

// parseA2AMessage extracts a workflow name and inputs from an A2A message.
// Supported format: JSON text part {"workflow": "name", "inputs": {"key": "value"}}
func parseA2AMessage(msg *a2a.Message) (string, map[string]any, error) {
	if msg == nil || len(msg.Parts) == 0 {
		return "", nil, fmt.Errorf("empty message")
	}

	// Extract text from first TextPart.
	var text string
	for _, part := range msg.Parts {
		if tp, ok := part.(a2a.TextPart); ok {
			text = tp.Text
			break
		}
	}
	if text == "" {
		return "", nil, fmt.Errorf("no text content in message")
	}

	// Try JSON format.
	var structured struct {
		Workflow string         `json:"workflow"`
		Inputs   map[string]any `json:"inputs"`
	}
	if err := json.Unmarshal([]byte(text), &structured); err == nil && structured.Workflow != "" {
		return structured.Workflow, structured.Inputs, nil
	}

	// Try metadata-based workflow name.
	if msg.Metadata != nil {
		if wfName, ok := msg.Metadata["workflow"].(string); ok && wfName != "" {
			return wfName, nil, nil
		}
	}

	return "", nil, fmt.Errorf("could not determine workflow; send JSON: {\"workflow\": \"name\", \"inputs\": {...}}")
}

// buildAgentCard generates a dynamic AgentCard reflecting current workflows.
func (s *Server) buildAgentCard(ctx context.Context) *a2a.AgentCard {
	workflows, _ := s.repo.List(ctx)

	skills := make([]a2a.AgentSkill, 0, len(workflows))
	for _, wf := range workflows {
		var inputIDs []string
		for _, nd := range wf.Nodes {
			if nd.Type == upal.NodeTypeInput {
				inputIDs = append(inputIDs, nd.ID)
			}
		}

		description := fmt.Sprintf("Execute workflow %q", wf.Name)
		if len(inputIDs) > 0 {
			description += fmt.Sprintf(" (inputs: %s)", strings.Join(inputIDs, ", "))
		}

		example := fmt.Sprintf(`{"workflow": "%s", "inputs": {%s}}`,
			wf.Name, buildExampleInputs(inputIDs))

		skills = append(skills, a2a.AgentSkill{
			ID:          wf.Name,
			Name:        wf.Name,
			Description: description,
			Tags:        []string{"workflow", "upal"},
			Examples:    []string{example},
		})
	}

	return &a2a.AgentCard{
		Name:               "Upal",
		Description:        "Visual AI workflow platform. Each skill represents a saved workflow.",
		URL:                s.a2aBaseURL + "/a2a",
		Version:            "0.2.0",
		ProtocolVersion:    "0.2",
		DefaultInputModes:  []string{"application/json", "text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Capabilities:       a2a.AgentCapabilities{Streaming: true},
		Skills:             skills,
	}
}

func buildExampleInputs(inputIDs []string) string {
	parts := make([]string, len(inputIDs))
	for i, id := range inputIDs {
		parts[i] = fmt.Sprintf(`"%s": "..."`, id)
	}
	return strings.Join(parts, ", ")
}

// setupA2ARoutes registers A2A protocol endpoints on the Chi router.
func (s *Server) setupA2ARoutes(r chi.Router) {
	executor := &upalA2AExecutor{
		workflowSvc: s.workflowSvc,
	}

	reqHandler := a2asrv.NewHandler(executor)

	// Dynamic agent card — regenerated on every request to reflect current workflows.
	cardProducer := a2asrv.AgentCardProducerFn(func(ctx context.Context) (*a2a.AgentCard, error) {
		return s.buildAgentCard(ctx), nil
	})
	r.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewAgentCardHandler(cardProducer))

	// JSON-RPC endpoint for A2A protocol.
	r.Handle("/a2a", a2asrv.NewJSONRPCHandler(reqHandler))
}
