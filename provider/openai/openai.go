// Package openai implements an OpenAI-compatible LLM provider for the graft framework.
// It works with OpenAI API, OpenRouter, Ollama, LM Studio, and any OpenAI-compatible endpoint.
package openai

import (
	"bufio"
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

const defaultBaseURL = "https://api.openai.com/v1"
const defaultModel = "gpt-4o"

// Client is an OpenAI-compatible LLM provider.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	headers    map[string]string
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *Client) {
		c.apiKey = key
	}
}

// WithBaseURL sets a custom base URL (for OpenRouter, Ollama, LM Studio, etc.).
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

// WithHeader adds a custom HTTP header to all requests.
func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// New creates a new OpenAI-compatible client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		model:      defaultModel,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		headers:    make(map[string]string),
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

type openAIMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

type openAIToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function openAIFunctionCall  `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAITool struct {
	Type     string           `json:"type"`
	Function openAIFunctionDef `json:"function"`
}

type openAIFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openAIResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Choices []openAIChoice   `json:"choices"`
	Usage   openAIUsage      `json:"usage"`
}

type openAIChoice struct {
	Index        int            `json:"index"`
	Message      openAIMessage  `json:"message"`
	FinishReason string         `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIStreamDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason"`
}

type openAIStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Choices []openAIStreamChoice `json:"choices"`
	Usage   *openAIUsage         `json:"usage,omitempty"`
}

// convertMessages converts graft messages to OpenAI format.
func convertMessages(msgs []graft.Message) []openAIMessage {
	out := make([]openAIMessage, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case graft.RoleTool:
			if m.ToolResult != nil {
				content := ""
				switch v := m.ToolResult.Content.(type) {
				case string:
					content = v
				default:
					b, _ := json.Marshal(v)
					content = string(b)
				}
				out = append(out, openAIMessage{
					Role:       "tool",
					Content:    content,
					ToolCallID: m.ToolResult.CallID,
				})
			}
		case graft.RoleAssistant:
			msg := openAIMessage{
				Role:    "assistant",
				Content: m.Content,
			}
			if len(m.ToolCalls) > 0 {
				tcs := make([]openAIToolCall, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					tcs[i] = openAIToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: openAIFunctionCall{
							Name:      tc.Name,
							Arguments: string(tc.Arguments),
						},
					}
				}
				msg.ToolCalls = tcs
			}
			out = append(out, msg)
		default:
			out = append(out, openAIMessage{
				Role:    string(m.Role),
				Content: m.Content,
			})
		}
	}
	return out
}

// convertTools converts graft tool definitions to OpenAI format.
func convertTools(tools []graft.ToolDefinition) []openAITool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]openAITool, len(tools))
	for i, t := range tools {
		out[i] = openAITool{
			Type: "function",
			Function: openAIFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Schema,
			},
		}
	}
	return out
}

// doRequest builds and executes an HTTP POST request to the completions endpoint.
func (c *Client) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: http request: %w", err)
	}
	return resp, nil
}

// Generate sends a non-streaming generation request.
func (c *Client) Generate(ctx context.Context, params graft.GenerateParams) (*graft.GenerateResult, error) {
	reqBody := openAIRequest{
		Model:       c.model,
		Messages:    convertMessages(params.Messages),
		Tools:       convertTools(params.Tools),
		Temperature: params.Temperature,
		MaxTokens:   params.MaxTokens,
		Stop:        params.Stop,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, b)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, graft.NewProviderError(resp.StatusCode, "openai", respBytes)
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("openai: unmarshal response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices in response")
	}

	choice := apiResp.Choices[0].Message
	msg := graft.Message{
		Role:    graft.RoleAssistant,
		Content: choice.Content,
	}

	if len(choice.ToolCalls) > 0 {
		tcs := make([]graft.ToolCall, len(choice.ToolCalls))
		for i, tc := range choice.ToolCalls {
			tcs[i] = graft.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: json.RawMessage(tc.Function.Arguments),
			}
		}
		msg.ToolCalls = tcs
	}

	return &graft.GenerateResult{
		Message: msg,
		Usage: graft.Usage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
		},
	}, nil
}

// Stream sends a streaming generation request and returns a channel of StreamChunk.
func (c *Client) Stream(ctx context.Context, params graft.GenerateParams) (<-chan graft.StreamChunk, error) {
	reqBody := openAIRequest{
		Model:       c.model,
		Messages:    convertMessages(params.Messages),
		Tools:       convertTools(params.Tools),
		Temperature: params.Temperature,
		MaxTokens:   params.MaxTokens,
		Stop:        params.Stop,
		Stream:      true,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, b)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, graft.NewProviderError(resp.StatusCode, "openai", body)
	}

	ch := make(chan graft.StreamChunk)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk openAIStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta
			event := graft.StreamEvent{
				Type:      graft.EventTextDelta,
				Data:      delta.Content,
				Timestamp: time.Now(),
			}

			sc := graft.StreamChunk{Delta: event}
			if chunk.Usage != nil {
				u := graft.Usage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
				}
				sc.Usage = &u
			}

			select {
			case ch <- sc:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}
