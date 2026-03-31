package bedrock

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/delavalom/graft"
)

// Stream sends a streaming generation request to the Bedrock ConverseStream API.
func (c *Client) Stream(ctx context.Context, params graft.GenerateParams) (<-chan graft.StreamChunk, error) {
	body, err := c.buildRequestBody(params)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, body, "converse-stream")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, graft.NewProviderError(resp.StatusCode, "bedrock", respBody)
	}

	ch := make(chan graft.StreamChunk)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		decoder := newEventStreamDecoder(resp.Body)
		toolBlocks := make(map[int]*streamToolUseStart)

		for {
			eventType, payload, err := decoder.readEvent()
			if err == io.EOF {
				break
			}
			if err != nil {
				ch <- graft.StreamChunk{
					Delta: graft.StreamEvent{
						Type:      graft.EventError,
						Data:      err.Error(),
						Timestamp: time.Now(),
					},
				}
				break
			}

			chunks := handleStreamEvent(eventType, payload, toolBlocks)
			for _, chunk := range chunks {
				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// handleStreamEvent processes a single event stream event and returns the
// corresponding graft StreamChunks.
func handleStreamEvent(eventType string, payload []byte, toolBlocks map[int]*streamToolUseStart) []graft.StreamChunk {
	switch eventType {
	case "messageStart":
		// No chunks emitted for messageStart.
		return nil

	case "contentBlockStart":
		var evt streamContentBlockStart
		if err := json.Unmarshal(payload, &evt); err != nil {
			return nil
		}
		if evt.Start.ToolUse != nil {
			toolBlocks[evt.ContentBlockIndex] = evt.Start.ToolUse
			return []graft.StreamChunk{
				{
					Delta: graft.StreamEvent{
						Type: graft.EventToolCallStart,
						Data: map[string]string{
							"id":   evt.Start.ToolUse.ToolUseID,
							"name": evt.Start.ToolUse.Name,
						},
						Timestamp: time.Now(),
					},
				},
			}
		}
		return nil

	case "contentBlockDelta":
		var evt streamContentBlockDelta
		if err := json.Unmarshal(payload, &evt); err != nil {
			return nil
		}
		if evt.Delta.ToolUse != nil {
			return []graft.StreamChunk{
				{
					Delta: graft.StreamEvent{
						Type:      graft.EventToolCallDelta,
						Data:      evt.Delta.ToolUse.Input,
						Timestamp: time.Now(),
					},
				},
			}
		}
		if evt.Delta.Text != "" {
			return []graft.StreamChunk{
				{
					Delta: graft.StreamEvent{
						Type:      graft.EventTextDelta,
						Data:      evt.Delta.Text,
						Timestamp: time.Now(),
					},
				},
			}
		}
		return nil

	case "contentBlockStop":
		var evt streamContentBlockStop
		if err := json.Unmarshal(payload, &evt); err != nil {
			return nil
		}
		if _, ok := toolBlocks[evt.ContentBlockIndex]; ok {
			delete(toolBlocks, evt.ContentBlockIndex)
			return []graft.StreamChunk{
				{
					Delta: graft.StreamEvent{
						Type:      graft.EventToolCallDone,
						Timestamp: time.Now(),
					},
				},
			}
		}
		return nil

	case "messageStop":
		return []graft.StreamChunk{
			{
				Delta: graft.StreamEvent{
					Type:      graft.EventMessageDone,
					Timestamp: time.Now(),
				},
			},
		}

	case "metadata":
		var evt streamMetadata
		if err := json.Unmarshal(payload, &evt); err != nil {
			return nil
		}
		usage := graft.Usage{
			PromptTokens:     evt.Usage.InputTokens,
			CompletionTokens: evt.Usage.OutputTokens,
		}
		return []graft.StreamChunk{
			{
				Delta: graft.StreamEvent{
					Type:      graft.EventDone,
					Timestamp: time.Now(),
				},
				Usage: &usage,
			},
		}

	default:
		return nil
	}
}
