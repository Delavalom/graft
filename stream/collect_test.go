package stream

import (
	"testing"
	"time"

	"github.com/delavalom/graft"
)

func TestCollect(t *testing.T) {
	ch := make(chan graft.StreamEvent, 3)
	ch <- graft.StreamEvent{Type: graft.EventTextDelta, Data: "Hello", Timestamp: time.Now()}
	ch <- graft.StreamEvent{Type: graft.EventTextDelta, Data: " world", Timestamp: time.Now()}
	ch <- graft.StreamEvent{Type: graft.EventDone, Timestamp: time.Now()}
	close(ch)

	result, err := Collect(ch)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.LastAssistantText() != "Hello world" {
		t.Errorf("LastAssistantText() = %q, want %q", result.LastAssistantText(), "Hello world")
	}
}

func TestCollectError(t *testing.T) {
	ch := make(chan graft.StreamEvent, 2)
	ch <- graft.StreamEvent{Type: graft.EventError, Data: "something broke", Timestamp: time.Now()}
	ch <- graft.StreamEvent{Type: graft.EventDone, Timestamp: time.Now()}
	close(ch)

	_, err := Collect(ch)
	if err == nil {
		t.Fatal("expected error from Collect")
	}
}

func TestCollectEmpty(t *testing.T) {
	ch := make(chan graft.StreamEvent, 1)
	ch <- graft.StreamEvent{Type: graft.EventDone, Timestamp: time.Now()}
	close(ch)

	result, err := Collect(ch)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.LastAssistantText() != "" {
		t.Errorf("expected empty text, got %q", result.LastAssistantText())
	}
}
