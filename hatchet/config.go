package hatchet

import (
	"context"
	"time"
)

// Option configures a HatchetRunner.
type Option func(*runnerConfig)

type runnerConfig struct {
	namespace   string
	concurrency int
	rateLimits  map[string]int
}

// WithNamespace sets the Hatchet namespace.
func WithNamespace(ns string) Option {
	return func(c *runnerConfig) { c.namespace = ns }
}

// WithConcurrency sets the max concurrent task runs.
func WithConcurrency(n int) Option {
	return func(c *runnerConfig) { c.concurrency = n }
}

// WithRateLimit adds a rate limit for a given key.
func WithRateLimit(key string, limit int) Option {
	return func(c *runnerConfig) {
		if c.rateLimits == nil {
			c.rateLimits = make(map[string]int)
		}
		c.rateLimits[key] = limit
	}
}

// RetryConfig configures retry behavior for Hatchet tasks.
type RetryConfig struct {
	MaxAttempts    int
	BackoffFactor  float64
	MaxBackoffSecs int
	NonRetryable   []string // error types that should not be retried
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:    3,
		BackoffFactor:  2.0,
		MaxBackoffSecs: 60,
	}
}

// IsNonRetryable checks if an error type is marked non-retryable.
func (c *RetryConfig) IsNonRetryable(errType string) bool {
	for _, t := range c.NonRetryable {
		if t == errType {
			return true
		}
	}
	return false
}

// ConcurrencyConfig controls concurrency for tasks.
type ConcurrencyConfig struct {
	MaxRuns   int
	Strategy  ConcurrencyStrategy
	GroupKey  string // for GROUP_ROUND_ROBIN
}

// ConcurrencyStrategy defines how concurrent tasks are handled.
type ConcurrencyStrategy string

const (
	StrategyCancel         ConcurrencyStrategy = "CANCEL"
	StrategyGroupRoundRobin ConcurrencyStrategy = "GROUP_ROUND_ROBIN"
	StrategyQueue          ConcurrencyStrategy = "QUEUE"
)

// RateLimitConfig defines rate limits.
type RateLimitConfig struct {
	Key      string
	Limit    int
	Duration time.Duration
	Dynamic  bool // per-user dynamic limits
}

// HatchetClient is the interface that a Hatchet client must satisfy.
type HatchetClient interface {
	RunTask(ctx context.Context, taskName string, input any) (TaskRun, error)
	RunWorkflow(ctx context.Context, workflowName string, input any) (WorkflowRun, error)
}

// TaskRun represents a running Hatchet task.
type TaskRun interface {
	ID() string
	Wait(ctx context.Context) (any, error)
}

// WorkflowRun represents a running Hatchet workflow.
type WorkflowRun interface {
	ID() string
	Wait(ctx context.Context) (any, error)
}

// TaskToolOption configures a TaskAsTool conversion.
type TaskToolOption func(*taskToolConfig)

type taskToolConfig struct {
	retryConfig *RetryConfig
}

// WithTaskRetry sets the retry config for a task-as-tool.
func WithTaskRetry(cfg *RetryConfig) TaskToolOption {
	return func(c *taskToolConfig) { c.retryConfig = cfg }
}
