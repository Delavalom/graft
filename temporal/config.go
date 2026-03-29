package temporal

import (
	"context"
	"time"
)

// Option configures a TemporalRunner.
type Option func(*runnerConfig)

type runnerConfig struct {
	taskQueue   string
	workflowID  string
	retryPolicy *RetryPolicy
}

// WithTaskQueue sets the Temporal task queue.
func WithTaskQueue(queue string) Option {
	return func(c *runnerConfig) { c.taskQueue = queue }
}

// WithWorkflowID sets an explicit workflow ID.
func WithWorkflowID(id string) Option {
	return func(c *runnerConfig) { c.workflowID = id }
}

// WithRetryPolicy sets the retry policy for activities.
func WithRetryPolicy(p *RetryPolicy) Option {
	return func(c *runnerConfig) { c.retryPolicy = p }
}

// RetryPolicy configures retry behavior for Temporal activities.
type RetryPolicy struct {
	MaxAttempts        int
	InitialInterval    time.Duration
	BackoffCoefficient float64
	MaxInterval        time.Duration
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:        3,
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaxInterval:        30 * time.Second,
	}
}

// WorkflowConfig holds workflow-level settings.
type WorkflowConfig struct {
	MaxIterations int
	Timeout       time.Duration
	TaskQueue     string
	RetryPolicy   *RetryPolicy
}

// ActivityConfig holds activity-level settings.
type ActivityConfig struct {
	StartToCloseTimeout    time.Duration
	ScheduleToCloseTimeout time.Duration
	HeartbeatTimeout       time.Duration
	RetryPolicy            *RetryPolicy
}

// DefaultActivityConfig returns sensible defaults for activity execution.
func DefaultActivityConfig() *ActivityConfig {
	return &ActivityConfig{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         DefaultRetryPolicy(),
	}
}

// TemporalClient is the interface that a Temporal SDK client must satisfy.
// Users provide their own *client.Client that satisfies this interface.
type TemporalClient interface {
	ExecuteWorkflow(ctx context.Context, options WorkflowOptions, workflow string, args ...any) (WorkflowRun, error)
	QueryWorkflow(ctx context.Context, workflowID, runID, queryType string, args ...any) (EncodedValue, error)
	SignalWorkflow(ctx context.Context, workflowID, runID, signalName string, arg any) error
}

// WorkflowOptions mirrors temporal's client.StartWorkflowOptions.
type WorkflowOptions struct {
	ID        string
	TaskQueue string
}

// WorkflowRun represents a running Temporal workflow.
type WorkflowRun interface {
	GetID() string
	GetRunID() string
	Get(ctx context.Context, valuePtr any) error
}

// EncodedValue represents an encoded value from Temporal queries.
type EncodedValue interface {
	Get(valuePtr any) error
}

// WorkflowContext is the interface for Temporal's workflow.Context.
// Used inside workflows to execute activities and handle signals.
type WorkflowContext interface {
	Done() <-chan struct{}
}

// ActivityToolOption configures an ActivityAsTool conversion.
type ActivityToolOption func(*activityToolConfig)

type activityToolConfig struct {
	activityConfig *ActivityConfig
}

// WithActivityConfig sets the activity config for an activity-as-tool.
func WithActivityConfig(cfg *ActivityConfig) ActivityToolOption {
	return func(c *activityToolConfig) { c.activityConfig = cfg }
}
