package trigger

import "time"

// Option configures a TriggerRunner.
type Option func(*runnerConfig)

type runnerConfig struct {
	apiKey      string
	projectID   string
	environment string
	pollInterval time.Duration
}

// WithAPIKey sets the Trigger.dev API key.
func WithAPIKey(key string) Option {
	return func(c *runnerConfig) { c.apiKey = key }
}

// WithProjectID sets the Trigger.dev project ID.
func WithProjectID(id string) Option {
	return func(c *runnerConfig) { c.projectID = id }
}

// WithEnvironment sets the Trigger.dev environment (dev, staging, prod).
func WithEnvironment(env string) Option {
	return func(c *runnerConfig) { c.environment = env }
}

// WithPollInterval sets the polling interval for run status checks.
func WithPollInterval(d time.Duration) Option {
	return func(c *runnerConfig) { c.pollInterval = d }
}

// TaskConfig holds task-level settings.
type TaskConfig struct {
	Priority int               `json:"priority,omitempty"`
	Queue    string            `json:"queue,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RunStatus represents the status of a task run.
type RunStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"` // "PENDING", "RUNNING", "COMPLETED", "FAILED", "CANCELLED"
	Output any    `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// RunEvent is an event from a task run subscription.
type RunEvent struct {
	Type string `json:"type"` // "STATUS_UPDATE", "OUTPUT", "LOG", "ERROR"
	Data any    `json:"data"`
}

// StreamConfig configures streaming settings.
type StreamConfig struct {
	BufferSize     int
	ReconnectDelay time.Duration
	MaxReconnects  int
}

// DefaultStreamConfig returns sensible defaults.
func DefaultStreamConfig() *StreamConfig {
	return &StreamConfig{
		BufferSize:     64,
		ReconnectDelay: time.Second,
		MaxReconnects:  5,
	}
}
