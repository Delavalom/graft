package temporal

// This file contains the workflow and activity function signatures.
// In production, users register these with their Temporal worker.
// The actual workflow logic runs inside Temporal's deterministic sandbox.

// AgentWorkflowFunc is the signature for the agent workflow function.
// Users register this with their Temporal worker:
//
//	w.RegisterWorkflowWithOptions(temporal.AgentWorkflowFunc, ...)
//
// The workflow:
//  1. Receives WorkflowInput with agent config and messages
//  2. Loops up to MaxIterations:
//     a. Calls GenerateActivity to get LLM response
//     b. If response has tool calls, executes each via ToolActivity
//     c. Appends results to messages and continues
//  3. Returns WorkflowOutput with final messages and usage
//
// For handoffs, the workflow uses ContinueAsNew with the new agent's config.
// For human-in-the-loop, the workflow listens for signals on an "approval" channel.
type AgentWorkflowFunc = func(ctx interface{}, input WorkflowInput) (WorkflowOutput, error)

// GenerateActivityInput is the input for the LLM generation activity.
type GenerateActivityInput struct {
	Model       string          `json:"model"`
	Messages    []byte          `json:"messages"`    // JSON-encoded []graft.Message
	Tools       []byte          `json:"tools"`       // JSON-encoded []graft.ToolDefinition
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
}

// GenerateActivityOutput is the output from the LLM generation activity.
type GenerateActivityOutput struct {
	Message []byte `json:"message"` // JSON-encoded graft.Message
	Usage   []byte `json:"usage"`   // JSON-encoded graft.Usage
}

// ToolActivityInput is the input for a tool execution activity.
type ToolActivityInput struct {
	ToolName  string `json:"tool_name"`
	Arguments []byte `json:"arguments"` // JSON-encoded tool params
}

// ToolActivityOutput is the output from a tool execution activity.
type ToolActivityOutput struct {
	Result  []byte `json:"result"`   // JSON-encoded result
	IsError bool   `json:"is_error"`
}

// SignalApproval is the signal payload for human-in-the-loop approval.
type SignalApproval struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

// QueryCurrentState is the query name for getting current workflow state.
const QueryCurrentState = "current_state"

// SignalApprovalName is the signal name for approval workflows.
const SignalApprovalName = "approval"
