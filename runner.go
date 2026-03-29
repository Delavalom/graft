package graft

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Runner interface {
	Run(ctx context.Context, agent *Agent, messages []Message, opts ...RunOption) (*Result, error)
	RunStream(ctx context.Context, agent *Agent, messages []Message, opts ...RunOption) (<-chan StreamEvent, error)
}

type DefaultRunner struct {
	model LanguageModel
}

func NewDefaultRunner(model LanguageModel) *DefaultRunner {
	return &DefaultRunner{model: model}
}

func (r *DefaultRunner) Run(ctx context.Context, agent *Agent, messages []Message, opts ...RunOption) (*Result, error) {
	cfg := DefaultRunConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := ctx.Err(); err != nil {
		return nil, NewAgentError(ErrTimeout, "context cancelled before start", err)
	}

	// Fire agent_start hook
	if agent.Hooks != nil {
		_, err := agent.Hooks.Run(ctx, &HookPayload{Event: HookAgentStart, Agent: agent, Messages: messages})
		if err != nil {
			return nil, err
		}
	}

	// Grace context for cleanup
	graceCtx, graceCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer graceCancel()
	defer func() {
		if agent.Hooks != nil {
			agent.Hooks.Run(graceCtx, &HookPayload{Event: HookAgentEnd, Agent: agent, Messages: messages})
		}
	}()

	// Build tool map
	toolMap := r.buildToolMap(agent)
	toolDefs := r.buildToolDefs(toolMap)

	// Copy messages
	msgs := make([]Message, len(messages))
	copy(msgs, messages)

	// Prepend system message
	if agent.Instructions != "" {
		msgs = append([]Message{{Role: RoleSystem, Content: agent.Instructions}}, msgs...)
	}

	var totalUsage Usage
	activeAgent := agent

	for i := 0; i < cfg.MaxIterations; i++ {
		if err := ctx.Err(); err != nil {
			return nil, NewAgentError(ErrTimeout, "context cancelled during execution", err)
		}

		// Pre-generate hook
		if activeAgent.Hooks != nil {
			activeAgent.Hooks.Run(ctx, &HookPayload{Event: HookPreGenerate, Agent: activeAgent, Messages: msgs})
		}

		params := GenerateParams{
			Messages:    msgs,
			Tools:       toolDefs,
			Temperature: activeAgent.Temperature,
			MaxTokens:   activeAgent.MaxTokens,
			ToolChoice:  activeAgent.ToolChoice,
		}

		genResult, err := r.model.Generate(ctx, params)
		if err != nil {
			return nil, NewAgentError(ErrProvider, "model generation failed", err)
		}

		// Post-generate hook
		if activeAgent.Hooks != nil {
			activeAgent.Hooks.Run(ctx, &HookPayload{Event: HookPostGenerate, Agent: activeAgent, Messages: msgs})
		}

		totalUsage.PromptTokens += genResult.Usage.PromptTokens
		totalUsage.CompletionTokens += genResult.Usage.CompletionTokens

		msgs = append(msgs, genResult.Message)

		// No tool calls — done
		if len(genResult.Message.ToolCalls) == 0 {
			return &Result{Messages: msgs, Usage: totalUsage}, nil
		}

		// Execute tool calls
		toolResults, handoffAgent, err := r.executeToolCalls(ctx, activeAgent, toolMap, genResult.Message.ToolCalls, cfg.ParallelTools)
		if err != nil {
			return nil, err
		}

		for _, tr := range toolResults {
			msgs = append(msgs, Message{Role: RoleTool, ToolResult: &tr})
		}

		// Handle handoff
		if handoffAgent != nil {
			activeAgent = handoffAgent
			toolMap = r.buildToolMap(activeAgent)
			toolDefs = r.buildToolDefs(toolMap)
			if activeAgent.Instructions != "" && len(msgs) > 0 && msgs[0].Role == RoleSystem {
				msgs[0] = Message{Role: RoleSystem, Content: activeAgent.Instructions}
			}
		}
	}

	return nil, NewAgentError(ErrTimeout, fmt.Sprintf("max iterations (%d) exceeded", cfg.MaxIterations), nil)
}

func (r *DefaultRunner) RunStream(ctx context.Context, agent *Agent, messages []Message, opts ...RunOption) (<-chan StreamEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, NewAgentError(ErrTimeout, "context cancelled before start", err)
	}

	events := make(chan StreamEvent, 64)

	go func() {
		defer close(events)
		result, err := r.Run(ctx, agent, messages, opts...)
		if err != nil {
			events <- StreamEvent{Type: EventError, Data: err, AgentID: agent.Name, Timestamp: time.Now()}
			return
		}
		text := result.LastAssistantText()
		if text != "" {
			events <- StreamEvent{Type: EventTextDelta, Data: text, AgentID: agent.Name, Timestamp: time.Now()}
		}
		events <- StreamEvent{Type: EventDone, AgentID: agent.Name, Timestamp: time.Now()}
	}()

	return events, nil
}

func (r *DefaultRunner) buildToolMap(agent *Agent) map[string]Tool {
	m := make(map[string]Tool)
	for _, t := range agent.Tools {
		m[t.Name()] = t
	}
	for _, h := range agent.Handoffs {
		name := "handoff_" + h.Target.Name
		m[name] = &handoffTool{handoff: h, name: name}
	}
	return m
}

func (r *DefaultRunner) buildToolDefs(toolMap map[string]Tool) []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(toolMap))
	for _, t := range toolMap {
		defs = append(defs, ToolDefFromTool(t))
	}
	return defs
}

func (r *DefaultRunner) executeToolCalls(ctx context.Context, agent *Agent, toolMap map[string]Tool, calls []ToolCall, parallel bool) ([]ToolResult, *Agent, error) {
	results := make([]ToolResult, len(calls))
	var handoffAgent *Agent

	if parallel && len(calls) > 1 {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error

		for i, call := range calls {
			wg.Add(1)
			go func(idx int, tc ToolCall) {
				defer wg.Done()
				tr, ha, err := r.executeSingleTool(ctx, agent, toolMap, tc)
				mu.Lock()
				defer mu.Unlock()
				if err != nil && firstErr == nil {
					firstErr = err
				}
				results[idx] = tr
				if ha != nil {
					handoffAgent = ha
				}
			}(i, call)
		}
		wg.Wait()
		if firstErr != nil {
			return nil, nil, firstErr
		}
	} else {
		for i, call := range calls {
			tr, ha, err := r.executeSingleTool(ctx, agent, toolMap, call)
			if err != nil {
				return nil, nil, err
			}
			results[i] = tr
			if ha != nil {
				handoffAgent = ha
			}
		}
	}

	return results, handoffAgent, nil
}

func (r *DefaultRunner) executeSingleTool(ctx context.Context, agent *Agent, toolMap map[string]Tool, call ToolCall) (ToolResult, *Agent, error) {
	tool, ok := toolMap[call.Name]
	if !ok {
		return ToolResult{CallID: call.ID, Content: fmt.Sprintf("unknown tool: %s", call.Name), IsError: true}, nil, nil
	}

	// Pre-tool hook
	if agent.Hooks != nil {
		hookResult, err := agent.Hooks.Run(ctx, &HookPayload{Event: HookPreToolCall, Agent: agent, ToolCall: &call})
		if err != nil {
			return ToolResult{CallID: call.ID, Content: err.Error(), IsError: true}, nil, nil
		}
		if hookResult != nil && hookResult.Allow != nil && !*hookResult.Allow {
			return ToolResult{CallID: call.ID, Content: "tool call denied by hook", IsError: true}, nil, nil
		}
	}

	output, err := tool.Execute(ctx, call.Arguments)

	// Check handoff
	if ht, ok := tool.(*handoffTool); ok && err == nil {
		return ToolResult{CallID: call.ID, Content: fmt.Sprintf("Handed off to %s", ht.handoff.Target.Name)}, ht.handoff.Target, nil
	}

	if err != nil {
		if agent.Hooks != nil {
			agent.Hooks.Run(ctx, &HookPayload{Event: HookToolCallError, Agent: agent, ToolCall: &call})
		}
		return ToolResult{CallID: call.ID, Content: err.Error(), IsError: true}, nil, nil
	}

	content := formatToolOutput(output)

	// Post-tool hook
	if agent.Hooks != nil {
		agent.Hooks.Run(ctx, &HookPayload{Event: HookPostToolCall, Agent: agent, ToolCall: &call})
	}

	return ToolResult{CallID: call.ID, Content: content}, nil, nil
}

func formatToolOutput(v any) any {
	switch val := v.(type) {
	case string:
		return val
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(data)
	}
}

type handoffTool struct {
	handoff Handoff
	name    string
}

func (h *handoffTool) Name() string           { return h.name }
func (h *handoffTool) Description() string     { return h.handoff.Description }
func (h *handoffTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (h *handoffTool) Execute(ctx context.Context, params json.RawMessage) (any, error) {
	return fmt.Sprintf("Handing off to %s", h.handoff.Target.Name), nil
}
