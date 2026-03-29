package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/delavalom/graft"
)

// TriggerRunner wraps graft agent execution via Trigger.dev's REST API.
type TriggerRunner struct {
	client *Client
	cfg    runnerConfig
}

// NewTriggerRunner creates a new Trigger.dev-backed runner.
func NewTriggerRunner(baseURL string, opts ...Option) *TriggerRunner {
	cfg := runnerConfig{
		environment:  "dev",
		pollInterval: 2 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &TriggerRunner{
		client: NewClient(baseURL, cfg.apiKey),
		cfg:    cfg,
	}
}

// triggerPayload is the payload sent to Trigger.dev.
type triggerPayload struct {
	AgentName    string          `json:"agent_name"`
	Instructions string          `json:"instructions"`
	Model        string          `json:"model"`
	Messages     []graft.Message `json:"messages"`
	MaxIterations int            `json:"max_iterations"`
}

// triggerResult is the parsed result from Trigger.dev.
type triggerResult struct {
	Messages []graft.Message `json:"messages"`
	Usage    graft.Usage     `json:"usage"`
}

// Run triggers a task and polls for completion.
func (r *TriggerRunner) Run(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (*graft.Result, error) {
	cfg := graft.DefaultRunConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	payload := triggerPayload{
		AgentName:    agent.Name,
		Instructions: agent.Instructions,
		Model:        agent.Model,
		Messages:     messages,
		MaxIterations: cfg.MaxIterations,
	}

	taskID := fmt.Sprintf("graft-agent-%s", agent.Name)
	runID, err := r.client.TriggerTask(ctx, taskID, payload)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "trigger: failed to start task", err)
	}

	// Poll for completion
	for {
		select {
		case <-ctx.Done():
			return nil, graft.NewAgentError(graft.ErrTimeout, "trigger: context cancelled while polling", ctx.Err())
		case <-time.After(r.cfg.pollInterval):
		}

		status, err := r.client.GetRunStatus(ctx, runID)
		if err != nil {
			return nil, graft.NewAgentError(graft.ErrProvider, "trigger: failed to get run status", err)
		}

		switch status.Status {
		case "COMPLETED":
			var result triggerResult
			data, err := json.Marshal(status.Output)
			if err != nil {
				return nil, graft.NewAgentError(graft.ErrProvider, "trigger: failed to marshal output", err)
			}
			if err := json.Unmarshal(data, &result); err != nil {
				return nil, graft.NewAgentError(graft.ErrProvider, "trigger: failed to parse output", err)
			}
			return &graft.Result{
				Messages: result.Messages,
				Usage:    result.Usage,
			}, nil

		case "FAILED":
			return nil, graft.NewAgentError(graft.ErrProvider, fmt.Sprintf("trigger: task failed: %s", status.Error), nil)

		case "CANCELLED":
			return nil, graft.NewAgentError(graft.ErrProvider, "trigger: task was cancelled", nil)

		// PENDING, RUNNING — continue polling
		}
	}
}

// RunStream triggers a task and streams events via SSE.
func (r *TriggerRunner) RunStream(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (<-chan graft.StreamEvent, error) {
	cfg := graft.DefaultRunConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	payload := triggerPayload{
		AgentName:    agent.Name,
		Instructions: agent.Instructions,
		Model:        agent.Model,
		Messages:     messages,
		MaxIterations: cfg.MaxIterations,
	}

	taskID := fmt.Sprintf("graft-agent-%s", agent.Name)
	runID, err := r.client.TriggerTask(ctx, taskID, payload)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "trigger: failed to start task", err)
	}

	events, err := r.client.SubscribeToRun(ctx, runID)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "trigger: failed to subscribe", err)
	}

	ch := make(chan graft.StreamEvent, 64)
	go func() {
		defer close(ch)
		for event := range events {
			ch <- graft.StreamEvent{
				Type:      mapEventType(event.Type),
				Data:      event.Data,
				Timestamp: time.Now(),
			}
		}
		ch <- graft.StreamEvent{
			Type:      graft.EventDone,
			Timestamp: time.Now(),
		}
	}()
	return ch, nil
}

// mapEventType converts Trigger.dev event types to graft stream event types.
func mapEventType(triggerType string) graft.EventType {
	switch triggerType {
	case "OUTPUT":
		return graft.EventTextDelta
	case "ERROR":
		return graft.EventError
	case "STATUS_UPDATE":
		return graft.EventMessageDone
	default:
		return graft.EventTextDelta
	}
}

// Ensure TriggerRunner satisfies graft.Runner.
var _ graft.Runner = (*TriggerRunner)(nil)
