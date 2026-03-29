// Package anthropic implements the Anthropic Messages API provider for the graft framework.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/delavalom/graft"
)

const (
	defaultBaseURL    = "https://api.anthropic.com/v1"
	defaultModel      = "claude-3-5-sonnet-20241022"
	defaultMaxTokens  = 4096
	anthropicVersion  = "2023-06-01"
)

// Client is an Anthropic Messages API provider.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *Client) {
		c.apiKey = key
	}
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithModel sets the model ID to use.
func WithModel(model string) Option {
	return func(c *Client) {
		c.model = model
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// New creates a new Anthropic client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		model:      defaultModel,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ModelID returns the model identifier.
func (c *Client) ModelID() string {
	return c.model
}

// --- Internal request/response types ---

type anthropicTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicToolUseBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type anthropicToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// anthropicContentBlock is used for unmarshaling response content blocks.
type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []any
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	MaxTokens int                `json:"max_tokens"`
	// Optional generation config
	Temperature *float64 `json:"temperature,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`
}

type anthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []anthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason"`
	StopSequence *string                 `json:"stop_sequence"`
	Usage        anthropicUsage          `json:"usage"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// convertMessages converts graft messages to Anthropic format.
// System messages are extracted separately; this returns (systemPrompt, messages).
func convertMessages(msgs []graft.Message) (string, []anthropicMessage) {
	var systemParts []string
	var out []anthropicMessage

	for _, m := range msgs {
		switch m.Role {
		case graft.RoleSystem:
			systemParts = append(systemParts, m.Content)
		case graft.RoleTool:
			if m.ToolResult != nil {
				block := anthropicToolResultBlock{
					Type:      "tool_result",
					ToolUseID: m.ToolResult.CallID,
					Content:   m.ToolResult.Content,
					IsError:   m.ToolResult.IsError,
				}
				out = append(out, anthropicMessage{
					Role:    "user",
					Content: []any{block},
				})
			}
		case graft.RoleAssistant:
			if len(m.ToolCalls) > 0 {
				blocks := make([]any, 0, len(m.ToolCalls)+1)
				if m.Content != "" {
					blocks = append(blocks, anthropicTextBlock{Type: "text", Text: m.Content})
				}
				for _, tc := range m.ToolCalls {
					blocks = append(blocks, anthropicToolUseBlock{
						Type:  "tool_use",
						ID:    tc.ID,
						Name:  tc.Name,
						Input: tc.Arguments,
					})
				}
				out = append(out, anthropicMessage{
					Role:    "assistant",
					Content: blocks,
				})
			} else {
				out = append(out, anthropicMessage{
					Role:    "assistant",
					Content: m.Content,
				})
			}
		default:
			out = append(out, anthropicMessage{
				Role:    string(m.Role),
				Content: m.Content,
			})
		}
	}

	system := strings.Join(systemParts, "\n")
	return system, out
}

// convertTools converts graft tool definitions to Anthropic format.
func convertTools(tools []graft.ToolDefinition) []anthropicTool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]anthropicTool, len(tools))
	for i, t := range tools {
		out[i] = anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Schema,
		}
	}
	return out
}

// doRequest builds and executes an HTTP POST request.
func (c *Client) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	url := c.baseURL + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic: http request: %w", err)
	}
	return resp, nil
}

// Generate sends a non-streaming generation request.
func (c *Client) Generate(ctx context.Context, params graft.GenerateParams) (*graft.GenerateResult, error) {
	system, messages := convertMessages(params.Messages)

	maxTokens := defaultMaxTokens
	if params.MaxTokens != nil {
		maxTokens = *params.MaxTokens
	}

	reqBody := anthropicRequest{
		Model:         c.model,
		Messages:      messages,
		System:        system,
		Tools:         convertTools(params.Tools),
		MaxTokens:     maxTokens,
		Temperature:   params.Temperature,
		StopSequences: params.Stop,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, b)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, graft.NewProviderError(resp.StatusCode, "anthropic", respBytes)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("anthropic: unmarshal response: %w", err)
	}

	msg := graft.Message{
		Role: graft.RoleAssistant,
	}

	var toolCalls []graft.ToolCall
	for _, block := range apiResp.Content {
		switch block.Type {
		case "text":
			msg.Content += block.Text
		case "tool_use":
			toolCalls = append(toolCalls, graft.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	return &graft.GenerateResult{
		Message: msg,
		Usage: graft.Usage{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
		},
	}, nil
}

// Stream sends a streaming generation request.
// For MVP this uses non-streaming and wraps in a channel.
func (c *Client) Stream(ctx context.Context, params graft.GenerateParams) (<-chan graft.StreamChunk, error) {
	result, err := c.Generate(ctx, params)
	if err != nil {
		return nil, err
	}

	ch := make(chan graft.StreamChunk, 1)
	go func() {
		defer close(ch)
		chunk := graft.StreamChunk{
			Delta: graft.StreamEvent{
				Type:      graft.EventTextDelta,
				Data:      result.Message.Content,
				Timestamp: time.Now(),
			},
			Usage: &result.Usage,
		}
		select {
		case ch <- chunk:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}
