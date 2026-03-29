package stream

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/delavalom/graft"
)

func TestSSEHandler(t *testing.T) {
	events := make(chan graft.StreamEvent, 3)
	events <- graft.StreamEvent{Type: graft.EventTextDelta, Data: "Hi", Timestamp: time.Now()}
	events <- graft.StreamEvent{Type: graft.EventDone, Timestamp: time.Now()}
	close(events)

	handler := SSEHandlerFromChannel(events)
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: text_delta") {
		t.Errorf("body missing text_delta event:\n%s", body)
	}
	if !strings.Contains(body, "event: done") {
		t.Errorf("body missing done event:\n%s", body)
	}
}

func TestSSEEventFormat(t *testing.T) {
	ev := graft.StreamEvent{Type: graft.EventTextDelta, Data: "hello"}
	line := formatSSEEvent(ev)
	if !strings.HasPrefix(line, "event: text_delta\n") {
		t.Errorf("unexpected prefix: %q", line)
	}
	dataLine := strings.TrimPrefix(strings.Split(line, "\n")[1], "data: ")
	var m map[string]any
	if err := json.Unmarshal([]byte(dataLine), &m); err != nil {
		t.Errorf("data is not valid JSON: %v, line: %q", err, dataLine)
	}
}
