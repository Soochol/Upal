package chat

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/tools"
	"google.golang.org/genai"
)

type ChatRegistry struct {
	tools map[string]*ChatTool
	rules []Rule
}

type Rule struct {
	Page      string
	Condition func(ctx map[string]any) bool // nil means always match
	Tools     []string
}

func NewRegistry() *ChatRegistry {
	return &ChatRegistry{tools: make(map[string]*ChatTool)}
}

// Register adds a chat tool to the registry.
func (r *ChatRegistry) Register(t *ChatTool) {
	r.tools[t.Name] = t
}

// AddRule adds a resolution rule that maps a page+condition to a set of tool names.
func (r *ChatRegistry) AddRule(rule Rule) {
	r.rules = append(r.rules, rule)
}

// Resolve returns the chat tools available for the given page and context.
func (r *ChatRegistry) Resolve(page string, context map[string]any) []*ChatTool {
	names := make(map[string]bool)
	for _, rule := range r.rules {
		if rule.Page != page {
			continue
		}
		if rule.Condition != nil && !rule.Condition(context) {
			continue
		}
		for _, name := range rule.Tools {
			names[name] = true
		}
	}
	var result []*ChatTool
	for name := range names {
		if t, ok := r.tools[name]; ok {
			result = append(result, t)
		}
	}
	return result
}

// ToFunctionDeclarations converts chat tools to genai function declarations for the LLM.
func ToFunctionDeclarations(chatTools []*ChatTool) []*genai.FunctionDeclaration {
	decls := make([]*genai.FunctionDeclaration, len(chatTools))
	for i, t := range chatTools {
		decls[i] = &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  tools.ToGenaiSchema(t.Parameters),
		}
	}
	return decls
}

// ExecuteToolCall finds and executes a chat tool by name.
func (r *ChatRegistry) ExecuteToolCall(ctx context.Context, name string, args map[string]any) (any, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown chat tool: %s", name)
	}
	return t.Execute(ctx, args)
}
