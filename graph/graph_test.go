package graph

import (
	"context"
	"fmt"
	"testing"
)

// intState is a simple state for testing.
type intState struct {
	Value int
}

func TestLinearGraph(t *testing.T) {
	g := NewGraph[*intState]("linear")

	g.AddNode("a", func(_ context.Context, s *intState) (*intState, error) {
		s.Value += 1
		return s, nil
	})
	g.AddNode("b", func(_ context.Context, s *intState) (*intState, error) {
		s.Value *= 2
		return s, nil
	})
	g.AddNode("c", func(_ context.Context, s *intState) (*intState, error) {
		s.Value += 10
		return s, nil
	})

	g.SetEntryPoint(START)
	g.AddEdge(START, "a")
	g.AddEdge("a", "b")
	g.AddEdge("b", "c")
	g.SetEndPoint("c")

	cg, err := g.Compile()
	if err != nil {
		t.Fatal(err)
	}

	result, err := cg.Run(context.Background(), &intState{Value: 5})
	if err != nil {
		t.Fatal(err)
	}

	// (5 + 1) * 2 + 10 = 22
	if result.Value != 22 {
		t.Errorf("expected 22, got %d", result.Value)
	}
}

func TestConditionalRouting(t *testing.T) {
	g := NewGraph[*intState]("conditional")

	g.AddNode("check", func(_ context.Context, s *intState) (*intState, error) {
		return s, nil
	})
	g.AddNode("positive", func(_ context.Context, s *intState) (*intState, error) {
		s.Value = 100
		return s, nil
	})
	g.AddNode("negative", func(_ context.Context, s *intState) (*intState, error) {
		s.Value = -100
		return s, nil
	})

	g.SetEntryPoint(START)
	g.AddEdge(START, "check")
	g.AddConditionalEdge("check", func(_ context.Context, s *intState) string {
		if s.Value > 0 {
			return "pos"
		}
		return "neg"
	}, map[string]string{
		"pos": "positive",
		"neg": "negative",
	})
	g.SetEndPoint("positive", "negative")

	cg, err := g.Compile()
	if err != nil {
		t.Fatal(err)
	}

	// Positive path
	result, err := cg.Run(context.Background(), &intState{Value: 5})
	if err != nil {
		t.Fatal(err)
	}
	if result.Value != 100 {
		t.Errorf("expected 100, got %d", result.Value)
	}

	// Negative path
	result, err = cg.Run(context.Background(), &intState{Value: -5})
	if err != nil {
		t.Fatal(err)
	}
	if result.Value != -100 {
		t.Errorf("expected -100, got %d", result.Value)
	}
}

func TestMaxIterationsGuard(t *testing.T) {
	g := NewGraph[*intState]("loop")

	g.AddNode("loop", func(_ context.Context, s *intState) (*intState, error) {
		s.Value++
		return s, nil
	})

	g.SetEntryPoint(START)
	g.AddEdge(START, "loop")
	// Always loops back to itself — should hit max iterations
	g.AddConditionalEdge("loop", func(_ context.Context, _ *intState) string {
		return "again"
	}, map[string]string{
		"again": "loop",
	})

	cg, err := g.Compile()
	if err != nil {
		t.Fatal(err)
	}

	_, err = cg.Run(context.Background(), &intState{Value: 0})
	if err == nil {
		t.Fatal("expected max iterations error")
	}
}

func TestCompileValidation_NoEntryPoint(t *testing.T) {
	g := NewGraph[*intState]("no-entry")
	g.AddNode("a", func(_ context.Context, s *intState) (*intState, error) { return s, nil })
	g.AddEdge("a", END)

	_, err := g.Compile()
	if err == nil {
		t.Fatal("expected error for missing entry point")
	}
}

func TestCompileValidation_UnknownEdgeTarget(t *testing.T) {
	g := NewGraph[*intState]("bad-edge")
	g.AddNode("a", func(_ context.Context, s *intState) (*intState, error) { return s, nil })
	g.SetEntryPoint("a")
	g.AddEdge("a", "nonexistent")

	_, err := g.Compile()
	if err == nil {
		t.Fatal("expected error for unknown edge target")
	}
}

func TestCompileValidation_NoOutgoingEdges(t *testing.T) {
	g := NewGraph[*intState]("no-outgoing")
	g.AddNode("a", func(_ context.Context, s *intState) (*intState, error) { return s, nil })
	g.AddNode("b", func(_ context.Context, s *intState) (*intState, error) { return s, nil })
	g.SetEntryPoint(START)
	g.AddEdge(START, "a")
	g.AddEdge("a", "b")
	// b has no outgoing edge and is not an endpoint

	_, err := g.Compile()
	if err == nil {
		t.Fatal("expected error for node with no outgoing edges")
	}
}

func TestMessageState(t *testing.T) {
	ms := NewMessageState()

	ms.Set("key1", "value1")
	v, ok := ms.Get("key1")
	if !ok || v != "value1" {
		t.Errorf("expected value1, got %v (ok=%v)", v, ok)
	}

	_, ok = ms.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestStreamEvents(t *testing.T) {
	g := NewGraph[*intState]("stream")
	g.AddNode("a", func(_ context.Context, s *intState) (*intState, error) {
		s.Value = 42
		return s, nil
	})
	g.SetEntryPoint(START)
	g.AddEdge(START, "a")
	g.SetEndPoint("a")

	cg, err := g.Compile()
	if err != nil {
		t.Fatal(err)
	}

	ch, err := cg.RunStream(context.Background(), &intState{})
	if err != nil {
		t.Fatal(err)
	}

	var events []Event[*intState]
	for e := range ch {
		events = append(events, e)
	}

	// Expect: edge→a, node_start→a, node_end→a, done
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}

	foundStart := false
	foundEnd := false
	foundDone := false
	for _, e := range events {
		switch e.Type {
		case EventNodeStart:
			if e.NodeName == "a" {
				foundStart = true
			}
		case EventNodeEnd:
			if e.NodeName == "a" {
				foundEnd = true
			}
		case EventDone:
			foundDone = true
		}
	}
	if !foundStart {
		t.Error("missing node_start event for 'a'")
	}
	if !foundEnd {
		t.Error("missing node_end event for 'a'")
	}
	if !foundDone {
		t.Error("missing done event")
	}
}

func TestNodeError(t *testing.T) {
	g := NewGraph[*intState]("error")
	g.AddNode("fail", func(_ context.Context, s *intState) (*intState, error) {
		return s, fmt.Errorf("something went wrong")
	})
	g.SetEntryPoint(START)
	g.AddEdge(START, "fail")
	g.AddEdge("fail", END)

	cg, err := g.Compile()
	if err != nil {
		t.Fatal(err)
	}

	_, err = cg.Run(context.Background(), &intState{})
	if err == nil {
		t.Fatal("expected error from failing node")
	}
}

func TestContextCancellation(t *testing.T) {
	g := NewGraph[*intState]("cancel")
	g.AddNode("a", func(_ context.Context, s *intState) (*intState, error) {
		return s, nil
	})
	g.SetEntryPoint(START)
	g.AddEdge(START, "a")
	g.SetEndPoint("a")

	cg, err := g.Compile()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = cg.Run(ctx, &intState{})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
