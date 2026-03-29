package graft

import (
	"context"
	"sync"
)

// SubAgent represents a child agent that can be invoked as a tool by a parent agent.
type SubAgent struct {
	Agent       *Agent
	Description string
	// InputMapper optionally transforms parent messages before passing them to the child agent.
	InputMapper func([]Message) []Message
}

// SubAgentResult holds the result of a subagent execution.
type SubAgentResult struct {
	AgentName string
	Result    *Result
	Error     error
}

// RunSubAgent runs a single subagent with the provided messages.
// If the SubAgent has an InputMapper, it is applied to the messages before execution.
// Context isolation is enforced: the child agent receives its own message slice.
func RunSubAgent(ctx context.Context, runner Runner, sub *SubAgent, messages []Message) (*SubAgentResult, error) {
	childMessages := messages
	if sub.InputMapper != nil {
		childMessages = sub.InputMapper(messages)
	} else {
		// Defensive copy to ensure context isolation
		copied := make([]Message, len(messages))
		copy(copied, messages)
		childMessages = copied
	}

	result, err := runner.Run(ctx, sub.Agent, childMessages)
	return &SubAgentResult{
		AgentName: sub.Agent.Name,
		Result:    result,
		Error:     err,
	}, nil
}

// RunSubAgentsParallel runs multiple subagents concurrently and collects all results.
// All subagents are started simultaneously using a sync.WaitGroup.
// Errors from individual subagents are captured in their SubAgentResult and do not abort others.
func RunSubAgentsParallel(ctx context.Context, runner Runner, subs []*SubAgent, messages []Message) ([]*SubAgentResult, error) {
	results := make([]*SubAgentResult, len(subs))
	var wg sync.WaitGroup

	for i, sub := range subs {
		wg.Add(1)
		go func(idx int, s *SubAgent) {
			defer wg.Done()
			res, _ := RunSubAgent(ctx, runner, s, messages)
			results[idx] = res
		}(i, sub)
	}

	wg.Wait()
	return results, nil
}
