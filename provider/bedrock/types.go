// Package bedrock implements the AWS Bedrock Converse API provider for the graft framework.
package bedrock

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/delavalom/graft"
)

// --- Request types ---

type converseRequest struct {
	Messages        []converseMessage `json:"messages"`
	System          []systemBlock     `json:"system,omitempty"`
	InferenceConfig *inferenceConfig  `json:"inferenceConfig,omitempty"`
	ToolConfig      *toolConfig       `json:"toolConfig,omitempty"`
}

type systemBlock struct {
	Text string `json:"text"`
}

type converseMessage struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Text       string           `json:"text,omitempty"`
	ToolUse    *toolUseBlock    `json:"toolUse,omitempty"`
	ToolResult *toolResultBlock `json:"toolResult,omitempty"`
}

type toolUseBlock struct {
	ToolUseID string          `json:"toolUseId"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
}

type toolResultBlock struct {
	ToolUseID string         `json:"toolUseId"`
	Content   []contentBlock `json:"content"`
	Status    string         `json:"status"` // "success" or "error"
}

type inferenceConfig struct {
	MaxTokens     *int     `json:"maxTokens,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	StopSequences []string `json:"stopSequences,omitempty"`
}

type toolConfig struct {
	Tools      []toolDef   `json:"tools"`
	ToolChoice *toolChoice `json:"toolChoice,omitempty"`
}

type toolDef struct {
	ToolSpec toolSpec `json:"toolSpec"`
}

type toolSpec struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	JSON json.RawMessage `json:"json"`
}

type toolChoice struct {
	Auto *struct{} `json:"auto,omitempty"`
	Any  *struct{} `json:"any,omitempty"`
	Tool *struct {
		Name string `json:"name"`
	} `json:"tool,omitempty"`
}

// --- Response types ---

type converseResponse struct {
	Output    converseOutput `json:"output"`
	StopReason string       `json:"stopReason"`
	Usage     converseUsage  `json:"usage"`
}

type converseOutput struct {
	Message *converseMessage `json:"message,omitempty"`
}

type converseUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}

// --- ConverseStream event types ---

type streamMessageStart struct {
	Role string `json:"role"`
}

type streamContentBlockStart struct {
	ContentBlockIndex int `json:"contentBlockIndex"`
	Start             struct {
		ToolUse *streamToolUseStart `json:"toolUse,omitempty"`
	} `json:"start"`
}

type streamContentBlockMeta struct {
	ContentBlockIndex int    `json:"contentBlockIndex"`
	Type              string `json:"type"` // "text" or "tool_use"
}

type streamToolUseStart struct {
	ToolUseID string `json:"toolUseId"`
	Name      string `json:"name"`
}

type streamContentBlockDelta struct {
	ContentBlockIndex int         `json:"contentBlockIndex"`
	Delta             streamDelta `json:"delta"`
}

type streamDelta struct {
	Text    string          `json:"text,omitempty"`
	ToolUse *streamToolDelta `json:"toolUse,omitempty"`
}

type streamToolDelta struct {
	Input string `json:"input"` // partial JSON string
}

type streamContentBlockStop struct {
	ContentBlockIndex int `json:"contentBlockIndex"`
}

type streamMessageStop struct {
	StopReason string `json:"stopReason"`
}

type streamMetadata struct {
	Usage converseUsage `json:"usage"`
}

// --- Conversion functions ---

// convertMessages converts graft messages to Bedrock Converse API format.
// System messages are extracted into a separate slice of systemBlock.
func convertMessages(msgs []graft.Message) ([]systemBlock, []converseMessage) {
	var system []systemBlock
	var out []converseMessage

	for _, m := range msgs {
		switch m.Role {
		case graft.RoleSystem:
			system = append(system, systemBlock{Text: m.Content})

		case graft.RoleTool:
			if m.ToolResult != nil {
				status := "success"
				if m.ToolResult.IsError {
					status = "error"
				}

				// Marshal content to string
				var contentText string
				switch v := m.ToolResult.Content.(type) {
				case string:
					contentText = v
				default:
					b, _ := json.Marshal(v)
					contentText = string(b)
				}

				block := contentBlock{
					ToolResult: &toolResultBlock{
						ToolUseID: m.ToolResult.CallID,
						Content:   []contentBlock{{Text: contentText}},
						Status:    status,
					},
				}
				out = append(out, converseMessage{
					Role:    "user",
					Content: []contentBlock{block},
				})
			}

		case graft.RoleAssistant:
			var blocks []contentBlock
			if m.Content != "" {
				blocks = append(blocks, contentBlock{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, contentBlock{
					ToolUse: &toolUseBlock{
						ToolUseID: tc.ID,
						Name:      tc.Name,
						Input:     tc.Arguments,
					},
				})
			}
			if len(blocks) > 0 {
				out = append(out, converseMessage{
					Role:    "assistant",
					Content: blocks,
				})
			}

		default: // user
			out = append(out, converseMessage{
				Role:    string(m.Role),
				Content: []contentBlock{{Text: m.Content}},
			})
		}
	}

	return system, out
}

// convertTools converts graft tool definitions to Bedrock Converse API format.
func convertTools(tools []graft.ToolDefinition) []toolDef {
	if len(tools) == 0 {
		return nil
	}
	out := make([]toolDef, len(tools))
	for i, t := range tools {
		out[i] = toolDef{
			ToolSpec: toolSpec{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: inputSchema{JSON: t.Schema},
			},
		}
	}
	return out
}

// convertToolChoice converts a graft ToolChoice to Bedrock format.
// Returns nil for ToolChoiceNone (Bedrock has no "none" equivalent; omit toolConfig instead).
func convertToolChoice(tc graft.ToolChoice) *toolChoice {
	switch {
	case tc == graft.ToolChoiceAuto || tc == "":
		return &toolChoice{Auto: &struct{}{}}
	case tc == graft.ToolChoiceRequired:
		return &toolChoice{Any: &struct{}{}}
	case tc == graft.ToolChoiceNone:
		return nil
	default:
		// Check for specific tool: "specific:<name>"
		if name, ok := strings.CutPrefix(string(tc), "specific:"); ok {
			return &toolChoice{Tool: &struct {
				Name string `json:"name"`
			}{Name: name}}
		}
		return &toolChoice{Auto: &struct{}{}}
	}
}

// parseResponseMessage converts a Bedrock Converse response message to a graft Message.
func parseResponseMessage(msg *converseMessage) graft.Message {
	out := graft.Message{
		Role: graft.RoleAssistant,
	}
	if msg == nil {
		return out
	}

	var toolCalls []graft.ToolCall
	var textParts []string

	for _, block := range msg.Content {
		if block.Text != "" {
			textParts = append(textParts, block.Text)
		}
		if block.ToolUse != nil {
			toolCalls = append(toolCalls, graft.ToolCall{
				ID:        block.ToolUse.ToolUseID,
				Name:      block.ToolUse.Name,
				Arguments: block.ToolUse.Input,
			})
		}
	}

	out.Content = strings.Join(textParts, "")
	if len(toolCalls) > 0 {
		out.ToolCalls = toolCalls
	}

	return out
}

// bedrockEndpoint returns the Bedrock Converse API endpoint URL for the given region and model.
func bedrockEndpoint(region, modelID string) string {
	return fmt.Sprintf(
		"https://bedrock-runtime.%s.amazonaws.com/model/%s/converse",
		region, modelID,
	)
}
