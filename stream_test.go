package graft

import (
	"testing"
	"time"
)

func TestStreamEventTypes(t *testing.T) {
	events := []EventType{
		EventTextDelta,
		EventToolCallStart,
		EventToolCallDelta,
		EventToolCallDone,
		EventToolResultDone,
		EventMessageDone,
		EventHandoff,
		EventError,
		EventDone,
	}
	seen := make(map[EventType]bool)
	for _, e := range events {
		if seen[e] {
			t.Errorf("duplicate event type: %q", e)
		}
		seen[e] = true
		if e == "" {
			t.Error("event type is empty string")
		}
	}
	if len(events) != 9 {
		t.Errorf("expected 9 event types, got %d", len(events))
	}
}

func TestStreamEventConstruction(t *testing.T) {
	ev := StreamEvent{
		Type:      EventTextDelta,
		Data:      "hello",
		AgentID:   "agent-1",
		Timestamp: time.Now(),
	}
	if ev.Type != EventTextDelta {
		t.Errorf("Type = %q, want %q", ev.Type, EventTextDelta)
	}
	if ev.Data.(string) != "hello" {
		t.Errorf("Data = %v, want %q", ev.Data, "hello")
	}
}
