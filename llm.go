package graft

import "context"

// LanguageModel is the interface that all LLM providers must implement.
type LanguageModel interface {
	Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error)
	Stream(ctx context.Context, params GenerateParams) (<-chan StreamChunk, error)
	ModelID() string
}

// GenerateParams holds the parameters for a generation request.
type GenerateParams struct {
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	MaxTokens   *int             `json:"max_tokens,omitempty"`
	ToolChoice  ToolChoice       `json:"tool_choice,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
}

// GenerateResult holds the result of a generation request.
type GenerateResult struct {
	Message Message `json:"message"`
	Usage   Usage   `json:"usage"`
	Cost    *Cost   `json:"cost,omitempty"`
}

// StreamChunk holds a single chunk of a streaming response.
type StreamChunk struct {
	Delta StreamEvent `json:"delta"`
	Usage *Usage      `json:"usage,omitempty"`
}
