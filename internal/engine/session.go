package engine

import (
	"sync"
	"time"

	"github.com/soochol/upal/internal/a2atypes"
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
		ID: a2atypes.GenerateID("sess"), WorkflowID: workflowID,
		State: make(map[string]any), Artifacts: make(map[string][]a2atypes.Artifact),
		Status: SessionRunning,
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

// GetStateCopy returns a snapshot copy of the session state, safe for concurrent use.
func (m *SessionManager) GetStateCopy(sessionID string) map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return make(map[string]any)
	}
	cp := make(map[string]any, len(s.State))
	for k, v := range s.State {
		cp[k] = v
	}
	return cp
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

func (m *SessionManager) SetArtifacts(sessionID, nodeID string, artifacts []a2atypes.Artifact) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		s.Artifacts[nodeID] = artifacts
		s.UpdatedAt = time.Now()
	}
}

func (m *SessionManager) GetArtifacts(sessionID, nodeID string) []a2atypes.Artifact {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil
	}
	src := s.Artifacts[nodeID]
	if src == nil {
		return nil
	}
	cp := make([]a2atypes.Artifact, len(src))
	copy(cp, src)
	return cp
}

func (m *SessionManager) GetAllArtifacts(sessionID string) map[string][]a2atypes.Artifact {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil
	}
	cp := make(map[string][]a2atypes.Artifact, len(s.Artifacts))
	for k, v := range s.Artifacts {
		cp[k] = v
	}
	return cp
}

// GenerateID generates a random ID with the given prefix.
// Deprecated: Use a2atypes.GenerateID directly.
func GenerateID(prefix string) string {
	return a2atypes.GenerateID(prefix)
}
