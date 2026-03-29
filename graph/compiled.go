package graph

import (
	"context"
	"fmt"
	"time"
)

// EventType describes the type of a graph execution event.
type EventType string

const (
	EventNodeStart EventType = "node_start"
	EventNodeEnd   EventType = "node_end"
	EventEdge      EventType = "edge"
	EventError     EventType = "error"
	EventDone      EventType = "done"
)

// Event is emitted during graph execution for streaming.
type Event[S any] struct {
	Type      EventType `json:"type"`
	NodeName  string    `json:"node_name,omitempty"`
	State     S         `json:"state"`
	Timestamp time.Time `json:"timestamp"`
	Error     error     `json:"-"`
}

// CompiledGraph is a validated, ready-to-run graph.
type CompiledGraph[S any] struct {
	name       string
	nodes      map[string]NodeFunc[S]
	staticAdj  map[string][]string
	condAdj    map[string]conditionalEdge[S]
	entryPoint string
	endPoints  map[string]bool
	maxIter    int
}

// Run executes the graph to completion, returning the final state.
func (g *CompiledGraph[S]) Run(ctx context.Context, initial S) (S, error) {
	state := initial
	current := g.entryPoint

	// If entry is START, follow the edge from START
	if current == START {
		next, ok := g.nextNode(ctx, START, state)
		if !ok {
			return state, fmt.Errorf("graph %q: no edge from START", g.name)
		}
		current = next
	}

	for i := 0; i < g.maxIter; i++ {
		if err := ctx.Err(); err != nil {
			return state, err
		}

		if current == END || g.endPoints[current] {
			// If it's an actual node (not END sentinel), run it first
			if current != END {
				fn := g.nodes[current]
				var err error
				state, err = fn(ctx, state)
				if err != nil {
					return state, err
				}
			}
			return state, nil
		}

		fn, ok := g.nodes[current]
		if !ok {
			return state, fmt.Errorf("graph %q: node %q not found", g.name, current)
		}

		var err error
		state, err = fn(ctx, state)
		if err != nil {
			return state, fmt.Errorf("graph %q: node %q: %w", g.name, current, err)
		}

		next, ok := g.nextNode(ctx, current, state)
		if !ok {
			return state, fmt.Errorf("graph %q: no edge from node %q", g.name, current)
		}
		current = next
	}

	return state, fmt.Errorf("graph %q: max iterations (%d) exceeded", g.name, g.maxIter)
}

// RunStream executes the graph and streams events for each node.
func (g *CompiledGraph[S]) RunStream(ctx context.Context, initial S) (<-chan Event[S], error) {
	ch := make(chan Event[S], 16)
	go func() {
		defer close(ch)
		state := initial
		current := g.entryPoint

		if current == START {
			next, ok := g.nextNode(ctx, START, state)
			if !ok {
				ch <- Event[S]{Type: EventError, Error: fmt.Errorf("no edge from START"), Timestamp: time.Now()}
				return
			}
			ch <- Event[S]{Type: EventEdge, NodeName: next, State: state, Timestamp: time.Now()}
			current = next
		}

		for i := 0; i < g.maxIter; i++ {
			if err := ctx.Err(); err != nil {
				ch <- Event[S]{Type: EventError, Error: err, Timestamp: time.Now()}
				return
			}

			if current == END {
				ch <- Event[S]{Type: EventDone, State: state, Timestamp: time.Now()}
				return
			}

			if g.endPoints[current] {
				fn := g.nodes[current]
				ch <- Event[S]{Type: EventNodeStart, NodeName: current, State: state, Timestamp: time.Now()}
				var err error
				state, err = fn(ctx, state)
				if err != nil {
					ch <- Event[S]{Type: EventError, NodeName: current, Error: err, Timestamp: time.Now()}
					return
				}
				ch <- Event[S]{Type: EventNodeEnd, NodeName: current, State: state, Timestamp: time.Now()}
				ch <- Event[S]{Type: EventDone, State: state, Timestamp: time.Now()}
				return
			}

			fn, ok := g.nodes[current]
			if !ok {
				ch <- Event[S]{Type: EventError, Error: fmt.Errorf("node %q not found", current), Timestamp: time.Now()}
				return
			}

			ch <- Event[S]{Type: EventNodeStart, NodeName: current, State: state, Timestamp: time.Now()}
			var err error
			state, err = fn(ctx, state)
			if err != nil {
				ch <- Event[S]{Type: EventError, NodeName: current, Error: err, Timestamp: time.Now()}
				return
			}
			ch <- Event[S]{Type: EventNodeEnd, NodeName: current, State: state, Timestamp: time.Now()}

			next, ok := g.nextNode(ctx, current, state)
			if !ok {
				ch <- Event[S]{Type: EventError, Error: fmt.Errorf("no edge from %q", current), Timestamp: time.Now()}
				return
			}
			ch <- Event[S]{Type: EventEdge, NodeName: next, State: state, Timestamp: time.Now()}
			current = next
		}

		ch <- Event[S]{Type: EventError, Error: fmt.Errorf("max iterations exceeded"), Timestamp: time.Now()}
	}()
	return ch, nil
}

// nextNode determines the next node from the current one.
func (g *CompiledGraph[S]) nextNode(ctx context.Context, current string, state S) (string, bool) {
	// Check conditional edges first
	if ce, ok := g.condAdj[current]; ok {
		key := ce.router(ctx, state)
		if target, ok := ce.targets[key]; ok {
			return target, true
		}
		return "", false
	}

	// Check static edges
	if targets, ok := g.staticAdj[current]; ok && len(targets) > 0 {
		return targets[0], true
	}

	return "", false
}
