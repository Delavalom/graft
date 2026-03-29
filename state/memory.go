package state

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryStore is a thread-safe in-memory session store. Suitable for dev and testing.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*Session),
	}
}

// Save stores a deep copy of the session.
func (m *MemoryStore) Save(_ context.Context, session *Session) error {
	if session == nil {
		return fmt.Errorf("session must not be nil")
	}
	session.UpdatedAt = time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = cloneSession(session)
	return nil
}

// Load retrieves a session by ID, returning ErrNotFound if absent.
func (m *MemoryStore) Load(_ context.Context, sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, sessionID)
	}
	return cloneSession(s), nil
}

// List returns all sessions for the given agent name.
func (m *MemoryStore) List(_ context.Context, agentName string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Session
	for _, s := range m.sessions {
		if s.AgentName == agentName {
			out = append(out, cloneSession(s))
		}
	}
	return out, nil
}

// Delete removes a session, returning ErrNotFound if it does not exist.
func (m *MemoryStore) Delete(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[sessionID]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, sessionID)
	}
	delete(m.sessions, sessionID)
	return nil
}
