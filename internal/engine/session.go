package engine

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]*Session)}
}

func (m *SessionManager) Create(workflowID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	sess := &Session{
		ID: GenerateID("sess"), WorkflowID: workflowID,
		State: make(map[string]any), Status: SessionRunning,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.sessions[sess.ID] = sess
	return sess
}

func (m *SessionManager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

func (m *SessionManager) SetState(sessionID, key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		s.State[key] = value
		s.UpdatedAt = time.Now()
	}
}

func (m *SessionManager) SetStatus(sessionID string, status SessionStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		s.Status = status
		s.UpdatedAt = time.Now()
	}
}

// GenerateID generates a random ID with the given prefix.
func GenerateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}
