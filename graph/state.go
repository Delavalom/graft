package graph

import "github.com/delavalom/graft"

// StateWithMessages is the interface that graph states must implement
// when using AgentNode to embed graft agents in the graph.
type StateWithMessages interface {
	GetMessages() []graft.Message
	SetMessages([]graft.Message)
}

// MessageState is the default state implementation with messages and a key-value store.
type MessageState struct {
	Messages []graft.Message
	Values   map[string]any
}

// NewMessageState creates a new MessageState.
func NewMessageState() *MessageState {
	return &MessageState{
		Values: make(map[string]any),
	}
}

func (s *MessageState) GetMessages() []graft.Message { return s.Messages }

func (s *MessageState) SetMessages(msgs []graft.Message) { s.Messages = msgs }

// Get retrieves a value from the state.
func (s *MessageState) Get(key string) (any, bool) {
	v, ok := s.Values[key]
	return v, ok
}

// Set stores a value in the state.
func (s *MessageState) Set(key string, value any) {
	if s.Values == nil {
		s.Values = make(map[string]any)
	}
	s.Values[key] = value
}

// Checkpoint is a serializable snapshot of graph state at a node boundary.
type Checkpoint[S any] struct {
	NodeName string `json:"node_name"`
	State    S      `json:"state"`
	Step     int    `json:"step"`
}
