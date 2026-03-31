package agentworkflow

import "go.temporal.io/sdk/worker"

// RegisterAgentWorkflow registers DefaultAgentWorkflow with a Temporal worker.
func RegisterAgentWorkflow(w worker.Worker) {
	w.RegisterWorkflow(DefaultAgentWorkflow)
}

// RegisterAgentActivities registers GenerateActivity and ToolActivity with a Temporal worker.
// The model provider and tool registry are used by activities at execution time to look up
// the actual LanguageModel and Tool implementations by name.
func RegisterAgentActivities(w worker.Worker, models ModelProvider, tools *ToolRegistry) {
	a := NewActivities(models, tools)
	w.RegisterActivity(a.GenerateActivity)
	w.RegisterActivity(a.ToolActivity)
}
