package agentworkflow

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/delavalom/graft"
	graftemporal "github.com/delavalom/graft/temporal"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowState represents the current state of the agent workflow,
// queryable via Temporal's query mechanism.
type WorkflowState struct {
	Iteration    int    `json:"iteration"`
	AgentName    string `json:"agent_name"`
	ToolCallsRun int    `json:"tool_calls_run"`
	Status       string `json:"status"` // "running", "waiting_approval", "completed", "failed"
}

// DefaultAgentWorkflow is the Temporal workflow function for durable agent execution.
// Each LLM call and tool execution runs as a separate activity with its own
// retry policy, giving per-step durability. On worker crash, only the failed
// step reruns.
//
// Register with your Temporal worker:
//
//	w.RegisterWorkflow(agentworkflow.DefaultAgentWorkflow)
func DefaultAgentWorkflow(ctx workflow.Context, input graftemporal.WorkflowInput) (graftemporal.WorkflowOutput, error) {
	var zero graftemporal.WorkflowOutput

	// Query handler for current state
	state := WorkflowState{
		AgentName: input.AgentName,
		Status:    "running",
	}
	if err := workflow.SetQueryHandler(ctx, graftemporal.QueryCurrentState, func() (WorkflowState, error) {
		return state, nil
	}); err != nil {
		return zero, fmt.Errorf("failed to set query handler: %w", err)
	}

	// Build initial messages with system prompt
	msgs := make([]graft.Message, 0, len(input.Messages)+1)
	if input.Instructions != "" {
		msgs = append(msgs, graft.Message{Role: graft.RoleSystem, Content: input.Instructions})
	}
	msgs = append(msgs, input.Messages...)

	// Build tool definitions from tool names
	toolDefs := buildToolDefsFromInput(input)

	// Configure activity options
	generateTimeout := input.Config.GenerateTimeout
	if generateTimeout == 0 {
		generateTimeout = 2 * time.Minute
	}
	toolTimeout := input.Config.ToolTimeout
	if toolTimeout == 0 {
		toolTimeout = 30 * time.Second
	}

	generateOpts := workflow.ActivityOptions{
		StartToCloseTimeout: generateTimeout,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy:         convertRetryPolicy(input.Config.RetryPolicy),
	}
	toolOpts := workflow.ActivityOptions{
		StartToCloseTimeout: toolTimeout,
		HeartbeatTimeout:    15 * time.Second,
		RetryPolicy:         convertRetryPolicy(input.Config.RetryPolicy),
	}

	// Build approval set
	approvalSet := make(map[string]bool)
	for _, name := range input.Config.ApprovalRequiredFor {
		approvalSet[name] = true
	}

	var totalUsage graft.Usage
	maxIter := input.Config.MaxIterations
	if maxIter == 0 {
		maxIter = 10
	}

	// Agent loop — mirrors DefaultRunner.Run() in runner.go
	for i := 0; i < maxIter; i++ {
		state.Iteration = i

		// Serialize messages and tools for the activity boundary
		msgsJSON, err := json.Marshal(msgs)
		if err != nil {
			state.Status = "failed"
			return zero, fmt.Errorf("marshal messages: %w", err)
		}
		toolsJSON, err := json.Marshal(toolDefs)
		if err != nil {
			state.Status = "failed"
			return zero, fmt.Errorf("marshal tools: %w", err)
		}

		// Execute GenerateActivity — DURABLE CHECKPOINT
		generateCtx := workflow.WithActivityOptions(ctx, generateOpts)
		var genOutput graftemporal.GenerateActivityOutput
		err = workflow.ExecuteActivity(generateCtx, "GenerateActivity", graftemporal.GenerateActivityInput{
			Model:    input.Model,
			Messages: msgsJSON,
			Tools:    toolsJSON,
		}).Get(ctx, &genOutput)
		if err != nil {
			state.Status = "failed"
			return zero, fmt.Errorf("GenerateActivity failed (iteration %d): %w", i, err)
		}

		// Deserialize the assistant message and usage
		var assistantMsg graft.Message
		if err := json.Unmarshal(genOutput.Message, &assistantMsg); err != nil {
			state.Status = "failed"
			return zero, fmt.Errorf("unmarshal assistant message: %w", err)
		}
		var usage graft.Usage
		if err := json.Unmarshal(genOutput.Usage, &usage); err != nil {
			state.Status = "failed"
			return zero, fmt.Errorf("unmarshal usage: %w", err)
		}

		totalUsage.PromptTokens += usage.PromptTokens
		totalUsage.CompletionTokens += usage.CompletionTokens
		msgs = append(msgs, assistantMsg)

		// No tool calls — we're done
		if len(assistantMsg.ToolCalls) == 0 {
			state.Status = "completed"
			return graftemporal.WorkflowOutput{
				Messages: msgs,
				Usage:    totalUsage,
			}, nil
		}

		// Execute tool calls in parallel — each is a DURABLE CHECKPOINT
		toolCtx := workflow.WithActivityOptions(ctx, toolOpts)
		type toolFuture struct {
			future  workflow.Future
			callIdx int
			denied  bool
		}
		futures := make([]toolFuture, 0, len(assistantMsg.ToolCalls))

		for j, call := range assistantMsg.ToolCalls {
			// Human-in-the-loop: wait for approval if required
			if approvalSet[call.Name] {
				state.Status = "waiting_approval"
				approved := waitForApproval(ctx)
				state.Status = "running"
				if !approved {
					msgs = append(msgs, graft.Message{
						Role: graft.RoleTool,
						ToolResult: &graft.ToolResult{
							CallID:  call.ID,
							Content: "tool call denied by human reviewer",
							IsError: true,
						},
					})
					futures = append(futures, toolFuture{callIdx: j, denied: true})
					continue
				}
			}

			f := workflow.ExecuteActivity(toolCtx, "ToolActivity", graftemporal.ToolActivityInput{
				ToolName:  call.Name,
				Arguments: call.Arguments,
			})
			futures = append(futures, toolFuture{future: f, callIdx: j})
		}

		// Collect results from all tool futures. Futures are collected sequentially
		// but Temporal executes activities in parallel on the worker side. We must
		// never return early from this loop — every future must be collected to
		// ensure all tool results are appended to the conversation history.
		for _, tf := range futures {
			if tf.denied {
				state.ToolCallsRun++
				continue
			}

			var toolOut graftemporal.ToolActivityOutput
			if err := tf.future.Get(ctx, &toolOut); err != nil {
				call := assistantMsg.ToolCalls[tf.callIdx]
				msgs = append(msgs, graft.Message{
					Role: graft.RoleTool,
					ToolResult: &graft.ToolResult{
						CallID:  call.ID,
						Content: err.Error(),
						IsError: true,
					},
				})
				state.ToolCallsRun++
				continue
			}

			call := assistantMsg.ToolCalls[tf.callIdx]

			var content any
			if err := json.Unmarshal(toolOut.Result, &content); err != nil {
				content = string(toolOut.Result)
			}

			msgs = append(msgs, graft.Message{
				Role: graft.RoleTool,
				ToolResult: &graft.ToolResult{
					CallID:  call.ID,
					Content: content,
					IsError: toolOut.IsError,
				},
			})
			state.ToolCallsRun++
		}

		// Check for handoff — use ContinueAsNew to switch agent (first match wins)
		if handoff := detectHandoff(assistantMsg.ToolCalls, input.Handoffs); handoff != nil {
			newInput := graftemporal.WorkflowInput{
				AgentName:       handoff.AgentName,
				Instructions:    handoff.Instructions,
				Model:           handoff.Model,
				ToolDefinitions: handoff.ToolDefinitions,
				Messages:        msgs,
				Config:          input.Config,
				Handoffs:        input.Handoffs,
			}
			if newInput.Model == "" {
				newInput.Model = input.Model
			}
			return zero, workflow.NewContinueAsNewError(ctx, DefaultAgentWorkflow, newInput)
		}
	}

	state.Status = "failed"
	return zero, fmt.Errorf("max iterations (%d) exceeded", maxIter)
}

// detectHandoff checks if any tool call is a handoff and returns the matching config.
func detectHandoff(calls []graft.ToolCall, handoffs []graftemporal.HandoffConfig) *graftemporal.HandoffConfig {
	for _, call := range calls {
		if !strings.HasPrefix(call.Name, "handoff_") {
			continue
		}
		for i := range handoffs {
			if handoffs[i].ToolName == call.Name {
				return &handoffs[i]
			}
		}
	}
	return nil
}

// waitForApproval blocks the workflow (durably) until a SignalApproval signal arrives.
func waitForApproval(ctx workflow.Context) bool {
	ch := workflow.GetSignalChannel(ctx, graftemporal.SignalApprovalName)
	var approval graftemporal.SignalApproval
	ch.Receive(ctx, &approval)
	return approval.Approved
}

// buildToolDefsFromInput returns tool definitions for the LLM.
// Uses full ToolDefinitions (with descriptions and schemas) when available,
// falling back to bare names for backward compatibility.
func buildToolDefsFromInput(input graftemporal.WorkflowInput) []graft.ToolDefinition {
	defs := make([]graft.ToolDefinition, 0, len(input.ToolDefinitions)+len(input.Handoffs))

	if len(input.ToolDefinitions) > 0 {
		defs = append(defs, input.ToolDefinitions...)
	} else {
		// Fallback: ToolNames only (no descriptions/schemas)
		for _, name := range input.ToolNames {
			defs = append(defs, graft.ToolDefinition{
				Name:   name,
				Schema: json.RawMessage(`{"type":"object"}`),
			})
		}
	}

	for _, h := range input.Handoffs {
		defs = append(defs, graft.ToolDefinition{
			Name:        h.ToolName,
			Description: fmt.Sprintf("Hand off to %s", h.AgentName),
			Schema:      json.RawMessage(`{"type":"object"}`),
		})
	}
	return defs
}

// convertRetryPolicy converts graft's temporal RetryPolicy to Temporal SDK's.
func convertRetryPolicy(p *graftemporal.RetryPolicy) *sdktemporal.RetryPolicy {
	if p == nil {
		return nil
	}
	return &sdktemporal.RetryPolicy{
		MaximumAttempts:    int32(p.MaxAttempts),
		InitialInterval:    p.InitialInterval,
		BackoffCoefficient: p.BackoffCoefficient,
		MaximumInterval:    p.MaxInterval,
	}
}
