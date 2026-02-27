package chat

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/generate"
	"github.com/soochol/upal/internal/upal"
)

// WorkflowDeps holds the dependencies needed by workflow chat tools.
type WorkflowDeps struct {
	Generator *generate.Generator
}

// RegisterWorkflowTools registers workflow-related chat tools and rules on the registry.
func RegisterWorkflowTools(reg *ChatRegistry, deps WorkflowDeps) {
	registerConfigureNode(reg, deps)
	registerGenerateWorkflow(reg, deps)
	registerAddNode(reg)
	registerRemoveNode(reg)
	registerListNodes(reg)

	// Base tools available on any workflows page.
	reg.AddRule(Rule{
		Page:  "workflows",
		Tools: []string{"generate_workflow", "add_node", "remove_node", "list_nodes"},
	})
	// Additional tool when a node is selected.
	reg.AddRule(Rule{
		Page:      "workflows",
		Condition: func(ctx map[string]any) bool { _, ok := ctx["selected_node_id"]; return ok },
		Tools:     []string{"configure_node"},
	})
}

// registerConfigureNode registers the configure_node tool which configures a
// selected node using the LLM.
func registerConfigureNode(reg *ChatRegistry, deps WorkflowDeps) {
	reg.Register(&ChatTool{
		Name:        "configure_node",
		Description: "Configure a selected workflow node based on a user instruction. The node's current state (type, config, label, description, upstream nodes) is read from the chat context.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"node_id": map[string]any{
					"type":        "string",
					"description": "ID of the node to configure",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "User instruction describing how to configure the node",
				},
			},
			"required": []any{"node_id", "message"},
		},
		Execute: func(ctx context.Context, args map[string]any) (any, error) {
			nodeID, _ := args["node_id"].(string)
			message, _ := args["message"].(string)
			if nodeID == "" || message == "" {
				return nil, fmt.Errorf("node_id and message are required")
			}

			chatCtx := GetChatContext(ctx)
			if chatCtx == nil {
				return nil, fmt.Errorf("missing chat context")
			}

			input := generate.ConfigureNodeInput{
				NodeID:  nodeID,
				Message: message,
			}

			// Extract node details from chat context.
			if sel, ok := chatCtx["selected_node"].(map[string]any); ok {
				input.NodeType, _ = sel["type"].(string)
				input.Label, _ = sel["label"].(string)
				input.Description, _ = sel["description"].(string)
				if cfg, ok := sel["config"].(map[string]any); ok {
					input.CurrentConfig = cfg
				}
			}

			// Extract upstream nodes from chat context.
			if upstreams, ok := chatCtx["upstream_nodes"].([]generate.UpstreamNodeInfo); ok {
				input.UpstreamNodes = upstreams
			}

			out, err := deps.Generator.ConfigureNode(ctx, input)
			if err != nil {
				return nil, fmt.Errorf("configure node: %w", err)
			}
			return out, nil
		},
	})
}

// registerGenerateWorkflow registers the generate_workflow tool which generates
// or edits a workflow from a natural-language description.
func registerGenerateWorkflow(reg *ChatRegistry, deps WorkflowDeps) {
	reg.Register(&ChatTool{
		Name:        "generate_workflow",
		Description: "Generate a new workflow or edit an existing one from a natural-language description.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"description": map[string]any{
					"type":        "string",
					"description": "Natural-language description of the workflow to generate or how to modify the existing one",
				},
			},
			"required": []any{"description"},
		},
		Execute: func(ctx context.Context, args map[string]any) (any, error) {
			description, _ := args["description"].(string)
			if description == "" {
				return nil, fmt.Errorf("description is required")
			}

			chatCtx := GetChatContext(ctx)

			// Extract existing workflow from context for edit mode.
			var existingWorkflow *upal.WorkflowDefinition
			if chatCtx != nil {
				if wf, ok := chatCtx["workflow"].(*upal.WorkflowDefinition); ok {
					existingWorkflow = wf
				}
			}

			wf, err := deps.Generator.Generate(ctx, description, existingWorkflow, nil)
			if err != nil {
				return nil, fmt.Errorf("generate workflow: %w", err)
			}
			return wf, nil
		},
	})
}

// registerAddNode registers the add_node tool which returns a node definition
// for the frontend to place on the canvas.
func registerAddNode(reg *ChatRegistry) {
	reg.Register(&ChatTool{
		Name:        "add_node",
		Description: "Add a new node to the workflow. Returns the node definition for the frontend to place on the canvas.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"node_type": map[string]any{
					"type":        "string",
					"enum":        []any{"input", "agent", "tool", "output"},
					"description": "Type of the node to add",
				},
				"label": map[string]any{
					"type":        "string",
					"description": "Display label for the node",
				},
				"config": map[string]any{
					"type":        "object",
					"description": "Optional initial configuration for the node",
				},
			},
			"required": []any{"node_type", "label"},
		},
		Execute: func(ctx context.Context, args map[string]any) (any, error) {
			nodeType, _ := args["node_type"].(string)
			label, _ := args["label"].(string)
			if nodeType == "" || label == "" {
				return nil, fmt.Errorf("node_type and label are required")
			}

			config := map[string]any{}
			if cfg, ok := args["config"].(map[string]any); ok {
				config = cfg
			}
			config["label"] = label

			node := upal.NodeDefinition{
				ID:     upal.GenerateID(nodeType),
				Type:   upal.NodeType(nodeType),
				Config: config,
			}
			return map[string]any{
				"action": "add_node",
				"node":   node,
			}, nil
		},
	})
}

// registerRemoveNode registers the remove_node tool which returns the node ID
// for the frontend to remove from the canvas.
func registerRemoveNode(reg *ChatRegistry) {
	reg.Register(&ChatTool{
		Name:        "remove_node",
		Description: "Remove a node from the workflow by its ID.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"node_id": map[string]any{
					"type":        "string",
					"description": "ID of the node to remove",
				},
			},
			"required": []any{"node_id"},
		},
		Execute: func(ctx context.Context, args map[string]any) (any, error) {
			nodeID, _ := args["node_id"].(string)
			if nodeID == "" {
				return nil, fmt.Errorf("node_id is required")
			}
			return map[string]any{
				"action":  "remove_node",
				"node_id": nodeID,
			}, nil
		},
	})
}

// registerListNodes registers the list_nodes tool which returns the current
// workflow nodes from the chat context.
func registerListNodes(reg *ChatRegistry) {
	reg.Register(&ChatTool{
		Name:        "list_nodes",
		Description: "List all nodes currently in the workflow.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Execute: func(ctx context.Context, args map[string]any) (any, error) {
			chatCtx := GetChatContext(ctx)
			if chatCtx == nil {
				return map[string]any{"nodes": []any{}}, nil
			}

			nodes, _ := chatCtx["nodes"]
			if nodes == nil {
				return map[string]any{"nodes": []any{}}, nil
			}
			return map[string]any{
				"action": "list_nodes",
				"nodes":  nodes,
			}, nil
		},
	})
}
