package graft

import "time"

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

func (u Usage) TotalTokens() int {
	return u.PromptTokens + u.CompletionTokens
}

type Cost struct {
	InputCostUSD  float64 `json:"input_cost_usd"`
	OutputCostUSD float64 `json:"output_cost_usd"`
}

func (c Cost) TotalUSD() float64 {
	return c.InputCostUSD + c.OutputCostUSD
}

type Span struct {
	Name       string         `json:"name"`
	StartTime  time.Time      `json:"start_time"`
	Duration   time.Duration  `json:"duration"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Children   []Span         `json:"children,omitempty"`
}

type Trace struct {
	AgentID   string    `json:"agent_id"`
	StartTime time.Time `json:"start_time"`
	Spans     []Span    `json:"spans"`
}

func NewTrace(agentID string) *Trace {
	return &Trace{
		AgentID:   agentID,
		StartTime: time.Now(),
	}
}

func (t *Trace) AddSpan(s Span) {
	t.Spans = append(t.Spans, s)
}

type Result struct {
	Messages []Message `json:"messages"`
	Usage    Usage     `json:"usage"`
	Cost     *Cost     `json:"cost,omitempty"`
	Trace    *Trace    `json:"trace,omitempty"`
}

func (r *Result) LastAssistantText() string {
	for i := len(r.Messages) - 1; i >= 0; i-- {
		if r.Messages[i].Role == RoleAssistant && r.Messages[i].Content != "" {
			return r.Messages[i].Content
		}
	}
	return ""
}
