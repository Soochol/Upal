package notify

import (
	"context"
	"fmt"
	"sync"

	"github.com/soochol/upal/internal/upal"
)

// Sender delivers messages to an external service.
type Sender interface {
	// Type returns the connection type this sender handles.
	Type() upal.ConnectionType
	// Send delivers a message using the resolved connection credentials.
	Send(ctx context.Context, conn *upal.Connection, message string) error
}

// SenderRegistry maps connection types to their senders.
type SenderRegistry struct {
	mu      sync.RWMutex
	senders map[upal.ConnectionType]Sender
}

func NewSenderRegistry() *SenderRegistry {
	return &SenderRegistry{senders: make(map[upal.ConnectionType]Sender)}
}

// Register adds a sender for a connection type.
func (r *SenderRegistry) Register(s Sender) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.senders[s.Type()] = s
}

// Get returns the sender for the given connection type.
func (r *SenderRegistry) Get(connType upal.ConnectionType) (Sender, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.senders[connType]
	if !ok {
		return nil, fmt.Errorf("no sender registered for connection type %q", connType)
	}
	return s, nil
}
