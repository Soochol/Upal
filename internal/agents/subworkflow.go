package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

const maxSubWorkflowDepth = 10

// SubWorkflowNodeBuilder creates agents that execute another saved workflow
// as a child DAG. It supports input_mapping for passing data from parent
// state to child inputs, and stores the child's final state back into
// the parent session state.
//
// Config:
//   - workflow_name: name of the target workflow to invoke
//   - input_mapping: map[string]string where keys are child input node IDs
//     and values are template strings resolved from parent state
type SubWorkflowNodeBuilder struct{}

func (b *SubWorkflowNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeSubWorkflow }

func (b *SubWorkflowNodeBuilder) Build(nd *upal.NodeDefinition, deps BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	workflowName, _ := nd.Config["workflow_name"].(string)

	if workflowName == "" {
		return nil, fmt.Errorf("subworkflow node %q: missing workflow_name in config", nodeID)
	}
	if deps.WfLookup == nil {
		return nil, fmt.Errorf("subworkflow node %q: no WorkflowLookup in BuildDeps", nodeID)
	}
	if deps.NodeRegistry == nil {
		return nil, fmt.Errorf("subworkflow node %q: no NodeRegistry in BuildDeps", nodeID)
	}

	// Parse input_mapping if present.
	var inputMapping map[string]string
	if raw, ok := nd.Config["input_mapping"].(map[string]any); ok {
		inputMapping = make(map[string]string, len(raw))
		for k, v := range raw {
			if s, ok := v.(string); ok {
				inputMapping[k] = s
			}
		}
	}

	wfLookup := deps.WfLookup
	nodeRegistry := deps.NodeRegistry

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Sub-workflow node %s → %s", nodeID, workflowName),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				parentState := ctx.Session().State()

				// 1. Cycle detection.
				stack := CallStackFromContext(ctx)
				if stack.Contains(workflowName) {
					yield(nil, fmt.Errorf("subworkflow node %q: cycle detected — %q is already in call stack %v", nodeID, workflowName, stack.Names))
					return
				}
				if len(stack.Names) >= maxSubWorkflowDepth {
					yield(nil, fmt.Errorf("subworkflow node %q: max depth %d exceeded", nodeID, maxSubWorkflowDepth))
					return
				}

				// 2. Look up the child workflow.
				childWf, err := wfLookup.Lookup(context.Background(), workflowName)
				if err != nil {
					yield(nil, fmt.Errorf("subworkflow node %q: lookup %q: %w", nodeID, workflowName, err))
					return
				}

				// 3. Build child DAG agent.
				childAgent, err := NewDAGAgent(childWf, nodeRegistry, deps)
				if err != nil {
					yield(nil, fmt.Errorf("subworkflow node %q: build child DAG: %w", nodeID, err))
					return
				}

				// 4. Resolve input mapping from parent state.
				childInputs := make(map[string]any)
				for childKey, tpl := range inputMapping {
					resolved := resolveTemplateFromState(tpl, parentState)
					childInputs[childKey] = resolved
				}

				// 5. Push updated call stack into context.
				newStack := &SubWorkflowCallStack{
					Names: append(append([]string{}, stack.Names...), workflowName),
				}
				childCtx := WithCallStack(ctx, newStack)
				_ = childCtx

				// 6. Run the child agent and collect events.
				// We iterate through the child agent's Run output, forwarding events.
				var lastEvent *session.Event
				for event, err := range childAgent.Run(ctx) {
					if err != nil {
						yield(nil, fmt.Errorf("subworkflow node %q: child error: %w", nodeID, err))
						return
					}
					lastEvent = event
				}

				// 7. Store child result in parent state.
				var result string
				if lastEvent != nil && lastEvent.Actions.StateDelta != nil {
					resultJSON, _ := json.Marshal(lastEvent.Actions.StateDelta)
					result = string(resultJSON)
				} else {
					result = "{}"
				}
				_ = parentState.Set(nodeID, result)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(result)},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = result
				yield(event, nil)
			}
		},
	})
}
