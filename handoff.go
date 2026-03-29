package graft

import "context"

type Handoff struct {
	Target      *Agent
	Description string
	Filter      func(ctx context.Context, messages []Message) bool
}
