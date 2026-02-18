package nodes

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/provider"
	"github.com/soochol/upal/internal/tools"
)

type testMockProvider struct{}

func (m *testMockProvider) Name() string { return "mock" }
func (m *testMockProvider) ChatCompletion(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		Content:      "Generated response about: " + req.Messages[len(req.Messages)-1].Content,
		FinishReason: "stop",
	}, nil
}
func (m *testMockProvider) ChatCompletionStream(ctx context.Context, req *provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	return nil, nil
}

func TestAgentNode_Execute_SimpleGenerate(t *testing.T) {
	providerReg := provider.NewRegistry()
	providerReg.Register(&testMockProvider{})
	toolReg := tools.NewRegistry()
	node := NewAgentNode(providerReg, toolReg, engine.NewEventBus())

	def := &engine.NodeDefinition{
		ID: "agent1", Type: engine.NodeTypeAgent,
		Config: map[string]any{"model": "mock/test", "prompt": "Tell me about {{input1}}"},
	}
	state := map[string]any{"input1": "AI trends"}
	result, err := node.Execute(context.Background(), def, state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("result type: got %T, want string", result)
	}
	if resultStr == "" {
		t.Error("result should not be empty")
	}
}

func TestAgentNode_Execute_WithSystemPrompt(t *testing.T) {
	providerReg := provider.NewRegistry()
	providerReg.Register(&testMockProvider{})
	toolReg := tools.NewRegistry()
	node := NewAgentNode(providerReg, toolReg, engine.NewEventBus())

	def := &engine.NodeDefinition{
		ID: "agent1", Type: engine.NodeTypeAgent,
		Config: map[string]any{"model": "mock/test", "system_prompt": "You are a researcher.", "prompt": "Research this topic"},
	}
	result, err := node.Execute(context.Background(), def, map[string]any{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == nil {
		t.Error("result should not be nil")
	}
}
