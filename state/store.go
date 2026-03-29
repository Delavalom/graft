package state

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/delavalom/graft"
)

// Session holds a persisted agent conversation.
type Session struct {
	ID        string            `json:"id"`
	AgentName string            `json:"agent_name"`
	Messages  []graft.Message   `json:"messages"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Store is the interface for session persistence backends.
type Store interface {
	Save(ctx context.Context, session *Session) error
	Load(ctx context.Context, sessionID string) (*Session, error)
	List(ctx context.Context, agentName string) ([]*Session, error)
	Delete(ctx context.Context, sessionID string) error
}

// ErrNotFound is returned when a session does not exist.
var ErrNotFound = fmt.Errorf("session not found")

// NewSession creates a new Session with a generated ID and timestamps set to now.
func NewSession(agentName string) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:        newID(),
		AgentName: agentName,
		Messages:  []graft.Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// newID generates a simple random hex ID without external dependencies.
func newID() string {
	b := make([]byte, 16)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = byte(r.Intn(256))
	}
	// Set version 4 and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
