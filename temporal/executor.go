package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/delavalom/graft"
)

// TemporalRunner wraps graft agent execution in Temporal workflows.
// It implements graft.Runner by starting a Temporal workflow for each Run call.
type TemporalRunner struct {
	client TemporalClient
	cfg    runnerConfig
}

// NewTemporalRunner creates a new Temporal-backed runner.
func NewTemporalRunner(client TemporalClient, opts ...Option) *TemporalRunner {
	cfg := runnerConfig{
		taskQueue:   "graft-agents",
		retryPolicy: DefaultRetryPolicy(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &TemporalRunner{client: client, cfg: cfg}
}

// WorkflowInput is the serializable input for the agent workflow.
type WorkflowInput struct {
	AgentName    string             `json:"agent_name"`
	Instructions string             `json:"instructions"`
	Model        string             `json:"model"`
	ToolNames    []string           `json:"tool_names"`
	Messages     []graft.Message    `json:"messages"`
	Config       WorkflowConfig     `json:"config"`
}

// WorkflowOutput is the serializable output from the agent workflow.
type WorkflowOutput struct {
	Messages []graft.Message `json:"messages"`
	Usage    graft.Usage     `json:"usage"`
}

// Run executes the agent loop as a Temporal workflow.
func (r *TemporalRunner) Run(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (*graft.Result, error) {
	cfg := graft.DefaultRunConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	toolNames := make([]string, len(agent.Tools))
	for i, t := range agent.Tools {
		toolNames[i] = t.Name()
	}

	input := WorkflowInput{
		AgentName:    agent.Name,
		Instructions: agent.Instructions,
		Model:        agent.Model,
		ToolNames:    toolNames,
		Messages:     messages,
		Config: WorkflowConfig{
			MaxIterations: cfg.MaxIterations,
			TaskQueue:     r.cfg.taskQueue,
			RetryPolicy:   r.cfg.retryPolicy,
		},
	}

	workflowID := r.cfg.workflowID
	if workflowID == "" {
		workflowID = fmt.Sprintf("graft-agent-%s-%d", agent.Name, time.Now().UnixNano())
	}

	wfOpts := WorkflowOptions{
		ID:        workflowID,
		TaskQueue: r.cfg.taskQueue,
	}

	run, err := r.client.ExecuteWorkflow(ctx, wfOpts, "AgentWorkflow", input)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "temporal: failed to start workflow", err)
	}

	var output WorkflowOutput
	if err := run.Get(ctx, &output); err != nil {
		return nil, graft.NewAgentError(graft.ErrProvider, "temporal: workflow execution failed", err)
	}

	return &graft.Result{
		Messages: output.Messages,
		Usage:    output.Usage,
	}, nil
}

// RunStream starts the workflow and streams events via Temporal queries.
func (r *TemporalRunner) RunStream(ctx context.Context, agent *graft.Agent, messages []graft.Message, opts ...graft.RunOption) (<-chan graft.StreamEvent, error) {
	ch := make(chan graft.StreamEvent, 16)

	go func() {
		defer close(ch)
		result, err := r.Run(ctx, agent, messages, opts...)
		if err != nil {
			ch <- graft.StreamEvent{
				Type:      graft.EventError,
				Data:      err.Error(),
				Timestamp: time.Now(),
			}
			return
		}

		// Emit the final result as a message done event
		if result != nil && len(result.Messages) > 0 {
			last := result.Messages[len(result.Messages)-1]
			ch <- graft.StreamEvent{
				Type:      graft.EventMessageDone,
				Data:      last.Content,
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

// Signal sends a signal to a running workflow (e.g., for human-in-the-loop).
func (r *TemporalRunner) Signal(ctx context.Context, workflowID, signalName string, data any) error {
	return r.client.SignalWorkflow(ctx, workflowID, "", signalName, data)
}

// Query queries a running workflow for state.
func (r *TemporalRunner) Query(ctx context.Context, workflowID, queryType string) (json.RawMessage, error) {
	val, err := r.client.QueryWorkflow(ctx, workflowID, "", queryType, nil)
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	if err := val.Get(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// Ensure TemporalRunner satisfies graft.Runner.
var _ graft.Runner = (*TemporalRunner)(nil)
