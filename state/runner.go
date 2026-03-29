package state

import (
	"context"

	"github.com/delavalom/graft"
)

// SessionRunner wraps a graft.Runner and transparently persists the conversation
// to a Store before and after each call.
type SessionRunner struct {
	inner     graft.Runner
	store     Store
	sessionID string
}

// NewSessionRunner creates a SessionRunner that loads/saves the session identified
// by sessionID using store, delegating actual generation to runner.
func NewSessionRunner(runner graft.Runner, store Store, sessionID string) *SessionRunner {
	return &SessionRunner{
		inner:     runner,
		store:     store,
		sessionID: sessionID,
	}
}

// Run loads any existing session messages, prepends them to the provided messages,
// delegates to the inner runner, and saves the resulting messages back to the store.
func (s *SessionRunner) Run(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (*graft.Result, error) {
	prior, err := s.loadMessages(ctx)
	if err != nil {
		return nil, err
	}

	combined := append(prior, messages...)
	result, err := s.inner.Run(ctx, agent, combined, opts...)
	if err != nil {
		return nil, err
	}

	if saveErr := s.saveMessages(ctx, agent.Name, result.Messages); saveErr != nil {
		return result, saveErr
	}
	return result, nil
}

// RunStream loads existing session messages, prepends them, streams via the inner
// runner, and saves the accumulated messages once the stream completes.
func (s *SessionRunner) RunStream(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (<-chan graft.StreamEvent, error) {
	prior, err := s.loadMessages(ctx)
	if err != nil {
		return nil, err
	}

	combined := append(prior, messages...)
	innerCh, err := s.inner.RunStream(ctx, agent, combined, opts...)
	if err != nil {
		return nil, err
	}

	out := make(chan graft.StreamEvent, 64)

	go func() {
		defer close(out)

		var accumulated []graft.Message
		var savedOnDone bool

		for ev := range innerCh {
			out <- ev

			switch ev.Type {
			case graft.EventDone:
				// After the stream is done we have no direct access to messages via
				// StreamEvent, so we persist what we know: the original combined input
				// plus a synthetic assistant message built from any text deltas.
				if !savedOnDone {
					savedOnDone = true
					_ = s.saveMessages(ctx, agent.Name, accumulated)
				}
			case graft.EventTextDelta:
				if text, ok := ev.Data.(string); ok {
					// Build up a synthetic assistant message from deltas.
					if len(accumulated) == 0 || accumulated[len(accumulated)-1].Role != graft.RoleAssistant {
						accumulated = append(combined, graft.Message{Role: graft.RoleAssistant, Content: text})
					} else {
						accumulated[len(accumulated)-1].Content += text
					}
				}
			}
		}
	}()

	return out, nil
}

// loadMessages retrieves previously stored messages for the session.
// Returns an empty slice when no session exists yet.
func (s *SessionRunner) loadMessages(ctx context.Context) ([]graft.Message, error) {
	session, err := s.store.Load(ctx, s.sessionID)
	if err != nil {
		// No existing session is fine — start fresh.
		return nil, nil
	}
	return session.Messages, nil
}

// saveMessages persists the current message list under the session ID.
func (s *SessionRunner) saveMessages(ctx context.Context, agentName string, messages []graft.Message) error {
	// Try to load the existing session so we preserve CreatedAt / Metadata.
	session, err := s.store.Load(ctx, s.sessionID)
	if err != nil {
		session = &Session{
			ID:        s.sessionID,
			AgentName: agentName,
		}
		now := session.UpdatedAt
		session.CreatedAt = now
	}

	session.Messages = messages
	return s.store.Save(ctx, session)
}
