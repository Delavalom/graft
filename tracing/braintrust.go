package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/delavalom/graft"
)

// BraintrustOption configures a BraintrustProvider.
type BraintrustOption func(*braintrustConfig)

type braintrustConfig struct {
	projectName    string
	experimentName string
	baseURL        string
}

// WithBraintrustProject sets the Braintrust project name.
func WithBraintrustProject(name string) BraintrustOption {
	return func(c *braintrustConfig) { c.projectName = name }
}

// WithBraintrustExperiment sets the Braintrust experiment name.
func WithBraintrustExperiment(name string) BraintrustOption {
	return func(c *braintrustConfig) { c.experimentName = name }
}

// WithBraintrustBaseURL sets the Braintrust API base URL.
func WithBraintrustBaseURL(url string) BraintrustOption {
	return func(c *braintrustConfig) { c.baseURL = url }
}

// BraintrustProvider exports traces to Braintrust.
type BraintrustProvider struct {
	apiKey string
	cfg    braintrustConfig
	client *http.Client
	batch  []braintrustSpanData
	mu     sync.Mutex
}

type braintrustSpanData struct {
	SpanType  string         `json:"span_type"`
	Name      string         `json:"name"`
	Input     any            `json:"input,omitempty"`
	Output    any            `json:"output,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	StartTime time.Time      `json:"start_time"`
	EndTime   time.Time      `json:"end_time,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// NewBraintrustProvider creates a new Braintrust tracing provider.
func NewBraintrustProvider(apiKey string, opts ...BraintrustOption) *BraintrustProvider {
	cfg := braintrustConfig{
		baseURL: "https://api.braintrust.dev/v1",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &BraintrustProvider{
		apiKey: apiKey,
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *BraintrustProvider) StartRun(ctx context.Context, info RunInfo) (context.Context, RunSpan) {
	span := &braintrustRunSpan{
		provider: p,
		data: braintrustSpanData{
			SpanType:  "task",
			Name:      info.AgentName,
			Input:     info.Messages,
			Metadata:  info.Metadata,
			StartTime: time.Now(),
		},
	}
	return ctx, span
}

func (p *BraintrustProvider) StartGeneration(ctx context.Context, info GenerationInfo) (context.Context, GenerationSpan) {
	span := &braintrustGenSpan{
		provider: p,
		data: braintrustSpanData{
			SpanType:  "llm",
			Name:      info.Model,
			Input:     info.Messages,
			StartTime: time.Now(),
		},
	}
	return ctx, span
}

func (p *BraintrustProvider) StartToolCall(ctx context.Context, info ToolCallInfo) (context.Context, ToolCallSpan) {
	span := &braintrustToolSpan{
		provider: p,
		data: braintrustSpanData{
			SpanType:  "tool",
			Name:      info.ToolName,
			Input:     info.Arguments,
			StartTime: time.Now(),
		},
	}
	return ctx, span
}

func (p *BraintrustProvider) addSpan(data braintrustSpanData) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.batch = append(p.batch, data)
}

func (p *BraintrustProvider) Flush(ctx context.Context) error {
	p.mu.Lock()
	if len(p.batch) == 0 {
		p.mu.Unlock()
		return nil
	}
	spans := p.batch
	p.batch = nil
	p.mu.Unlock()

	body, err := json.Marshal(spans)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.baseURL+"/spans", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("braintrust: flush: %w", err)
	}
	resp.Body.Close()
	return nil
}

type braintrustRunSpan struct {
	provider *BraintrustProvider
	data     braintrustSpanData
}

func (s *braintrustRunSpan) End(err error) {
	s.data.EndTime = time.Now()
	if err != nil {
		s.data.Error = err.Error()
	}
	s.provider.addSpan(s.data)
}

func (s *braintrustRunSpan) SetMetadata(meta map[string]any) { s.data.Metadata = meta }
func (s *braintrustRunSpan) SetUsage(usage graft.Usage) {
	if s.data.Metadata == nil {
		s.data.Metadata = make(map[string]any)
	}
	s.data.Metadata["prompt_tokens"] = usage.PromptTokens
	s.data.Metadata["completion_tokens"] = usage.CompletionTokens
}

type braintrustGenSpan struct {
	provider *BraintrustProvider
	data     braintrustSpanData
}

func (s *braintrustGenSpan) End(err error) {
	s.data.EndTime = time.Now()
	if err != nil {
		s.data.Error = err.Error()
	}
	s.provider.addSpan(s.data)
}

func (s *braintrustGenSpan) SetMetadata(meta map[string]any) { s.data.Metadata = meta }
func (s *braintrustGenSpan) SetModel(model string)           { s.data.Name = model }
func (s *braintrustGenSpan) SetUsage(usage graft.Usage) {
	if s.data.Metadata == nil {
		s.data.Metadata = make(map[string]any)
	}
	s.data.Metadata["prompt_tokens"] = usage.PromptTokens
	s.data.Metadata["completion_tokens"] = usage.CompletionTokens
}

type braintrustToolSpan struct {
	provider *BraintrustProvider
	data     braintrustSpanData
}

func (s *braintrustToolSpan) End(err error) {
	s.data.EndTime = time.Now()
	if err != nil {
		s.data.Error = err.Error()
	}
	s.provider.addSpan(s.data)
}

func (s *braintrustToolSpan) SetMetadata(meta map[string]any) { s.data.Metadata = meta }
func (s *braintrustToolSpan) SetResult(result any)             { s.data.Output = result }

var _ TracingProvider = (*BraintrustProvider)(nil)

// init suppresses unused import warning for log
func init() { _ = log.Prefix }
