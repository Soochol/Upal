package agents

import (
	"testing"

	"github.com/soochol/upal/internal/notify"
	"github.com/soochol/upal/internal/upal"
)

func TestApprovalNodeBuilder_Build(t *testing.T) {
	builder := &ApprovalNodeBuilder{}
	if builder.NodeType() != upal.NodeTypeApproval {
		t.Fatalf("NodeType() = %q, want %q", builder.NodeType(), upal.NodeTypeApproval)
	}

	sender := &mockSender{typ: upal.ConnTypeTelegram}
	senderReg := notify.NewSenderRegistry()
	senderReg.Register(sender)

	connResolver := &mockConnResolver{
		conns: map[string]*upal.Connection{
			"conn-tg": {
				ID:   "conn-tg",
				Type: upal.ConnTypeTelegram,
				Extras: map[string]any{
					"chat_id": "12345",
				},
			},
		},
	}

	nd := &upal.NodeDefinition{
		ID:   "approval1",
		Type: upal.NodeTypeApproval,
		Config: map[string]any{
			"connection_id": "conn-tg",
			"message":       "Please approve: {{agent1}}",
			"timeout":       float64(60),
		},
	}

	deps := BuildDeps{
		SenderReg:    senderReg,
		ConnResolver: connResolver,
	}

	ag, err := builder.Build(nd, deps)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if ag == nil {
		t.Fatal("Build() returned nil agent")
	}
}

func TestApprovalNodeBuilder_NoDeps(t *testing.T) {
	builder := &ApprovalNodeBuilder{}

	// Should still build without deps (notification step is optional).
	nd := &upal.NodeDefinition{
		ID:     "approval2",
		Type:   upal.NodeTypeApproval,
		Config: map[string]any{},
	}

	ag, err := builder.Build(nd, BuildDeps{})
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if ag == nil {
		t.Fatal("Build() returned nil agent")
	}
}

func TestApprovalNodeBuilder_DefaultTimeout(t *testing.T) {
	builder := &ApprovalNodeBuilder{}

	nd := &upal.NodeDefinition{
		ID:   "approval3",
		Type: upal.NodeTypeApproval,
		Config: map[string]any{
			"message": "approve this",
		},
	}

	ag, err := builder.Build(nd, BuildDeps{})
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if ag == nil {
		t.Fatal("Build() returned nil agent")
	}
}
