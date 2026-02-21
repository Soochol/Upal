package agents

import (
	"encoding/json"
	"fmt"
	"iter"
	"strings"

	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// IteratorNodeBuilder creates agents that iterate over an array source.
// For each item in the array it stores the item in session state under
// `item_key` (default: "<nodeID>_item") and emits an event.
// After all items are processed, it stores the collected results array
// in session state under the node ID.
//
// Config:
//   - source: template expression that resolves to a JSON array string,
//     e.g. "{{upstream_node}}" where upstream produced a JSON array
//   - item_key: session state key for the current item (default: "<nodeID>_item")
//   - max_iterations: maximum number of items to process (default: 100)
type IteratorNodeBuilder struct{}

func (b *IteratorNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeIterator }

func (b *IteratorNodeBuilder) Build(nd *upal.NodeDefinition, _ BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	sourceTpl, _ := nd.Config["source"].(string)
	itemKey, _ := nd.Config["item_key"].(string)
	if itemKey == "" {
		itemKey = nodeID + "_item"
	}
	maxIterations := 100
	if v, ok := nd.Config["max_iterations"].(float64); ok && int(v) > 0 {
		maxIterations = int(v)
	}

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Iterator node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				// Resolve the source template and parse as JSON array.
				resolved := resolveTemplateFromState(sourceTpl, state)
				items, err := parseJSONArray(resolved)
				if err != nil {
					yield(nil, fmt.Errorf("iterator node %q: parse source: %w", nodeID, err))
					return
				}

				// Limit iterations.
				if len(items) > maxIterations {
					items = items[:maxIterations]
				}

				// Process each item.
				var results []any
				for i, item := range items {
					// Store current item in session state.
					_ = state.Set(itemKey, item)
					_ = state.Set(nodeID+"_index", i)

					results = append(results, item)

					// Emit a per-item event.
					ev := session.NewEvent(ctx.InvocationID())
					ev.Author = nodeID
					ev.Branch = ctx.Branch()
					ev.LLMResponse = adkmodel.LLMResponse{
						Content: &genai.Content{
							Role:  "model",
							Parts: []*genai.Part{genai.NewPartFromText(fmt.Sprintf("item %d/%d", i+1, len(items)))},
						},
					}
					if !yield(ev, nil) {
						return
					}
				}

				// Store collected results.
				resultJSON, _ := json.Marshal(results)
				resultStr := string(resultJSON)
				_ = state.Set(nodeID, resultStr)

				// Emit final completion event.
				finalEv := session.NewEvent(ctx.InvocationID())
				finalEv.Author = nodeID
				finalEv.Branch = ctx.Branch()
				finalEv.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(resultStr)},
					},
					TurnComplete: true,
				}
				finalEv.Actions.StateDelta[nodeID] = resultStr
				yield(finalEv, nil)
			}
		},
	})
}

// parseJSONArray attempts to parse a string as a JSON array.
// It handles both JSON arrays and comma-separated plain strings.
func parseJSONArray(s string) ([]any, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return nil, nil
	}

	// Try JSON array first.
	if strings.HasPrefix(s, "[") {
		var arr []any
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			return arr, nil
		}
	}

	// Fallback: split by newlines (one item per line).
	lines := strings.Split(s, "\n")
	var items []any
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			items = append(items, line)
		}
	}
	return items, nil
}
