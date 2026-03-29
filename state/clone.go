package state

import "github.com/delavalom/graft"

// cloneSession returns a deep-enough copy of s so callers cannot mutate stored data.
func cloneSession(s *Session) *Session {
	cp := &Session{
		ID:        s.ID,
		AgentName: s.AgentName,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
	if s.Messages != nil {
		cp.Messages = make([]graft.Message, len(s.Messages))
		copy(cp.Messages, s.Messages)
	}
	if s.Metadata != nil {
		cp.Metadata = make(map[string]any, len(s.Metadata))
		for k, v := range s.Metadata {
			cp.Metadata[k] = v
		}
	}
	return cp
}
