package stream

import (
	"fmt"

	"github.com/delavalom/graft"
)

func Collect(events <-chan graft.StreamEvent) (*graft.Result, error) {
	var textParts []string
	var lastErr error

	for ev := range events {
		switch ev.Type {
		case graft.EventTextDelta:
			if s, ok := ev.Data.(string); ok {
				textParts = append(textParts, s)
			}
		case graft.EventError:
			switch v := ev.Data.(type) {
			case error:
				lastErr = v
			case string:
				lastErr = fmt.Errorf("%s", v)
			}
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	fullText := ""
	for _, p := range textParts {
		fullText += p
	}

	var messages []graft.Message
	if fullText != "" {
		messages = append(messages, graft.Message{
			Role:    graft.RoleAssistant,
			Content: fullText,
		})
	}

	return &graft.Result{Messages: messages}, nil
}
