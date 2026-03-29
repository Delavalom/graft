package stream

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/delavalom/graft"
)

func SSEHandlerFromChannel(events <-chan graft.StreamEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-events:
				if !ok {
					return
				}
				line := formatSSEEvent(ev)
				fmt.Fprint(w, line)
				flusher.Flush()
				if ev.Type == graft.EventDone {
					return
				}
			}
		}
	}
}

func formatSSEEvent(ev graft.StreamEvent) string {
	data, _ := json.Marshal(map[string]any{
		"type":     ev.Type,
		"data":     ev.Data,
		"agent_id": ev.AgentID,
	})
	return fmt.Sprintf("event: %s\ndata: %s\n\n", ev.Type, data)
}
