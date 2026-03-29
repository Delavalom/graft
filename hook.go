package graft

import "context"

type HookEvent string

const (
	HookAgentStart    HookEvent = "agent_start"
	HookAgentEnd      HookEvent = "agent_end"
	HookPreToolCall   HookEvent = "pre_tool_call"
	HookPostToolCall  HookEvent = "post_tool_call"
	HookToolCallError HookEvent = "tool_call_error"
	HookPreHandoff    HookEvent = "pre_handoff"
	HookPostHandoff   HookEvent = "post_handoff"
	HookPreGenerate   HookEvent = "pre_generate"
	HookPostGenerate  HookEvent = "post_generate"
	HookGuardrailTrip HookEvent = "guardrail_trip"
)

type HookPayload struct {
	Event    HookEvent
	Agent    *Agent
	ToolCall *ToolCall
	Messages []Message
	Metadata map[string]any
}

type HookResult struct {
	Allow         *bool
	ModifiedInput []byte
	AdditionalCtx string
	SkipExecution bool
}

type HookCallback func(ctx context.Context, payload *HookPayload) (*HookResult, error)

type HookRegistry struct {
	hooks map[HookEvent][]HookCallback
}

func NewHookRegistry() *HookRegistry {
	return &HookRegistry{hooks: make(map[HookEvent][]HookCallback)}
}

func (r *HookRegistry) On(event HookEvent, cb HookCallback) {
	r.hooks[event] = append(r.hooks[event], cb)
}

func (r *HookRegistry) Run(ctx context.Context, payload *HookPayload) (*HookResult, error) {
	if r == nil {
		return nil, nil
	}
	callbacks := r.hooks[payload.Event]
	for _, cb := range callbacks {
		result, err := cb(ctx, payload)
		if err != nil {
			return nil, err
		}
		if result != nil {
			if result.Allow != nil && !*result.Allow {
				return result, nil
			}
			if result.SkipExecution {
				return result, nil
			}
		}
	}
	return nil, nil
}
