// Package google implements the Google Generative Language (Gemini) API provider for graft.
package google

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
	defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	defaultModel   = "gemini-1.5-pro"
)

// Client is a Google Gemini API provider.
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

// New creates a new Google Gemini client with the given options.
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

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiRequest struct {
	Contents            []geminiContent          `json:"contents"`
	SystemInstruction   *geminiSystemInstruction `json:"systemInstruction,omitempty"`
	GenerationConfig    *geminiGenerationConfig  `json:"generationConfig,omitempty"`
}

type geminiResponsePart struct {
	Text string `json:"text,omitempty"`
}

type geminiResponseContent struct {
	Role  string               `json:"role"`
	Parts []geminiResponsePart `json:"parts"`
}

type geminiCandidate struct {
	Content       geminiResponseContent `json:"content"`
	FinishReason  string                `json:"finishReason"`
	Index         int                   `json:"index"`
	SafetyRatings []any                 `json:"safetyRatings,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate   `json:"candidates"`
	UsageMetadata geminiUsageMetadata `json:"usageMetadata"`
}

// convertMessages converts graft messages to Gemini format.
// System messages are extracted and returned separately.
func convertMessages(msgs []graft.Message) (string, []geminiContent) {
	var systemText string
	var out []geminiContent

	for _, m := range msgs {
		switch m.Role {
		case graft.RoleSystem:
			if systemText != "" {
				systemText += "\n"
			}
			systemText += m.Content
		case graft.RoleAssistant:
			out = append(out, geminiContent{
				Role:  "model",
				Parts: []geminiPart{{Text: m.Content}},
			})
		default:
			// user, tool, etc.
			content := m.Content
			if m.ToolResult != nil {
				switch v := m.ToolResult.Content.(type) {
				case string:
					content = v
				default:
					b, _ := json.Marshal(v)
					content = string(b)
				}
			}
			out = append(out, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: content}},
			})
		}
	}
	return systemText, out
}

// doRequest builds and executes an HTTP POST request to the generateContent endpoint.
func (c *Client) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("google: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google: http request: %w", err)
	}
	return resp, nil
}

// Generate sends a non-streaming generation request.
func (c *Client) Generate(ctx context.Context, params graft.GenerateParams) (*graft.GenerateResult, error) {
	systemText, contents := convertMessages(params.Messages)

	reqBody := geminiRequest{
		Contents: contents,
	}

	if systemText != "" {
		reqBody.SystemInstruction = &geminiSystemInstruction{
			Parts: []geminiPart{{Text: systemText}},
		}
	}

	if params.Temperature != nil || params.MaxTokens != nil || len(params.Stop) > 0 {
		reqBody.GenerationConfig = &geminiGenerationConfig{
			Temperature:     params.Temperature,
			MaxOutputTokens: params.MaxTokens,
			StopSequences:   params.Stop,
		}
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("google: marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, b)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("google: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, graft.NewProviderError(resp.StatusCode, "google", respBytes)
	}

	var apiResp geminiResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("google: unmarshal response: %w", err)
	}

	if len(apiResp.Candidates) == 0 {
		return nil, fmt.Errorf("google: no candidates in response")
	}

	candidate := apiResp.Candidates[0]
	var sb strings.Builder
	for _, part := range candidate.Content.Parts {
		sb.WriteString(part.Text)
	}
	text := sb.String()

	msg := graft.Message{
		Role:    graft.RoleAssistant,
		Content: text,
	}

	return &graft.GenerateResult{
		Message: msg,
		Usage: graft.Usage{
			PromptTokens:     apiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: apiResp.UsageMetadata.CandidatesTokenCount,
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
