package agents

import (
	"context"
	"fmt"
	"testing"

	"github.com/soochol/upal/internal/notify"
	"github.com/soochol/upal/internal/upal"
)

// mockSender records calls to Send.
type mockSender struct {
	typ     upal.ConnectionType
	calls   []mockSendCall
	sendErr error
}

type mockSendCall struct {
	ConnID  string
	Message string
}

func (m *mockSender) Type() upal.ConnectionType { return m.typ }

func (m *mockSender) Send(_ context.Context, conn *upal.Connection, message string) error {
	m.calls = append(m.calls, mockSendCall{ConnID: conn.ID, Message: message})
	return m.sendErr
}

// mockConnResolver implements ConnectionResolver for tests.
type mockConnResolver struct {
	conns map[string]*upal.Connection
}

func (r *mockConnResolver) Resolve(_ context.Context, id string) (*upal.Connection, error) {
	c, ok := r.conns[id]
	if !ok {
		return nil, fmt.Errorf("connection %q not found", id)
	}
	return c, nil
}

func TestNotificationNodeBuilder_Build(t *testing.T) {
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

	builder := &NotificationNodeBuilder{}
	if builder.NodeType() != upal.NodeTypeNotification {
		t.Fatalf("NodeType() = %q, want %q", builder.NodeType(), upal.NodeTypeNotification)
	}

	nd := &upal.NodeDefinition{
		ID:   "notify1",
		Type: upal.NodeTypeNotification,
		Config: map[string]any{
			"connection_id": "conn-tg",
			"message":       "Hello from {{input1}}!",
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

func TestNotificationNodeBuilder_NoDeps(t *testing.T) {
	builder := &NotificationNodeBuilder{}

	nd := &upal.NodeDefinition{
		ID:   "notify1",
		Type: upal.NodeTypeNotification,
		Config: map[string]any{
			"connection_id": "conn-tg",
			"message":       "hello",
		},
	}

	// No SenderReg
	_, err := builder.Build(nd, BuildDeps{})
	if err == nil {
		t.Fatal("expected error when SenderReg is nil")
	}

	// No ConnResolver
	senderReg := notify.NewSenderRegistry()
	_, err = builder.Build(nd, BuildDeps{SenderReg: senderReg})
	if err == nil {
		t.Fatal("expected error when ConnResolver is nil")
	}
}
