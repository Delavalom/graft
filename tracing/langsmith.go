package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/delavalom/graft"
)

// LangSmithOption configures a LangSmithProvider.
type LangSmithOption func(*langsmithConfig)

type langsmithConfig struct {
	projectName string
	baseURL     string
}

// WithLangSmithProject sets the LangSmith project name.
func WithLangSmithProject(name string) LangSmithOption {
	return func(c *langsmithConfig) { c.projectName = name }
}

// WithLangSmithBaseURL sets the LangSmith API base URL.
func WithLangSmithBaseURL(url string) LangSmithOption {
	return func(c *langsmithConfig) { c.baseURL = url }
}

// LangSmithProvider exports traces to LangSmith.
type LangSmithProvider struct {
	apiKey string
	cfg    langsmithConfig
	client *http.Client
	batch  []langsmithRunData
	mu     sync.Mutex
}

type langsmithRunData struct {
	Name      string         `json:"name"`
	RunType   string         `json:"run_type"` // "chain", "llm", "tool"
	Inputs    any            `json:"inputs,omitempty"`
	Outputs   any            `json:"outputs,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
	StartTime time.Time      `json:"start_time"`
	EndTime   time.Time      `json:"end_time,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// NewLangSmithProvider creates a new LangSmith tracing provider.
func NewLangSmithProvider(apiKey string, opts ...LangSmithOption) *LangSmithProvider {
	cfg := langsmithConfig{
		baseURL: "https://api.smith.langchain.com",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &LangSmithProvider{
		apiKey: apiKey,
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *LangSmithProvider) StartRun(ctx context.Context, info RunInfo) (context.Context, RunSpan) {
	span := &langsmithRunSpan{
		provider: p,
		data: langsmithRunData{
			Name:      info.AgentName,
			RunType:   "chain",
			Inputs:    info.Messages,
			Extra:     info.Metadata,
			StartTime: time.Now(),
		},
	}
	return ctx, span
}

func (p *LangSmithProvider) StartGeneration(ctx context.Context, info GenerationInfo) (context.Context, GenerationSpan) {
	span := &langsmithGenSpan{
		provider: p,
		data: langsmithRunData{
			Name:      info.Model,
			RunType:   "llm",
			Inputs:    info.Messages,
			StartTime: time.Now(),
		},
	}
	return ctx, span
}

func (p *LangSmithProvider) StartToolCall(ctx context.Context, info ToolCallInfo) (context.Context, ToolCallSpan) {
	span := &langsmithToolSpan{
		provider: p,
		data: langsmithRunData{
			Name:      info.ToolName,
			RunType:   "tool",
			Inputs:    info.Arguments,
			StartTime: time.Now(),
		},
	}
	return ctx, span
}

func (p *LangSmithProvider) addRun(data langsmithRunData) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.batch = append(p.batch, data)
}

func (p *LangSmithProvider) Flush(ctx context.Context) error {
	p.mu.Lock()
	if len(p.batch) == 0 {
		p.mu.Unlock()
		return nil
	}
	runs := p.batch
	p.batch = nil
	p.mu.Unlock()

	body, err := json.Marshal(runs)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.baseURL+"/runs/batch", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("langsmith: flush: %w", err)
	}
	resp.Body.Close()
	return nil
}

type langsmithRunSpan struct {
	provider *LangSmithProvider
	data     langsmithRunData
}

func (s *langsmithRunSpan) End(err error) {
	s.data.EndTime = time.Now()
	if err != nil {
		s.data.Error = err.Error()
	}
	s.provider.addRun(s.data)
}

func (s *langsmithRunSpan) SetMetadata(meta map[string]any) { s.data.Extra = meta }
func (s *langsmithRunSpan) SetUsage(usage graft.Usage) {
	if s.data.Extra == nil {
		s.data.Extra = make(map[string]any)
	}
	s.data.Extra["prompt_tokens"] = usage.PromptTokens
	s.data.Extra["completion_tokens"] = usage.CompletionTokens
}

type langsmithGenSpan struct {
	provider *LangSmithProvider
	data     langsmithRunData
}

func (s *langsmithGenSpan) End(err error) {
	s.data.EndTime = time.Now()
	if err != nil {
		s.data.Error = err.Error()
	}
	s.provider.addRun(s.data)
}

func (s *langsmithGenSpan) SetMetadata(meta map[string]any) { s.data.Extra = meta }
func (s *langsmithGenSpan) SetModel(model string)           { s.data.Name = model }
func (s *langsmithGenSpan) SetUsage(usage graft.Usage) {
	if s.data.Extra == nil {
		s.data.Extra = make(map[string]any)
	}
	s.data.Extra["prompt_tokens"] = usage.PromptTokens
	s.data.Extra["completion_tokens"] = usage.CompletionTokens
}

type langsmithToolSpan struct {
	provider *LangSmithProvider
	data     langsmithRunData
}

func (s *langsmithToolSpan) End(err error) {
	s.data.EndTime = time.Now()
	if err != nil {
		s.data.Error = err.Error()
	}
	s.provider.addRun(s.data)
}

func (s *langsmithToolSpan) SetMetadata(meta map[string]any) { s.data.Extra = meta }
func (s *langsmithToolSpan) SetResult(result any)             { s.data.Outputs = result }

var _ TracingProvider = (*LangSmithProvider)(nil)
