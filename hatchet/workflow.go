package hatchet

// DAGStep represents a step in a Hatchet DAG workflow.
type DAGStep struct {
	Name         string   `json:"name"`
	TaskName     string   `json:"task_name"`
	DependsOn    []string `json:"depends_on,omitempty"` // names of steps this depends on
	Concurrency  *ConcurrencyConfig `json:"concurrency,omitempty"`
	RetryConfig  *RetryConfig       `json:"retry_config,omitempty"`
}

// DAGWorkflow defines a Hatchet DAG for the agent loop.
type DAGWorkflow struct {
	Name  string    `json:"name"`
	Steps []DAGStep `json:"steps"`
}

// NewAgentDAG creates a standard agent loop DAG:
// generate → parse → execute_tools (fan-out) → collect → check_done
func NewAgentDAG(agentName string) *DAGWorkflow {
	return &DAGWorkflow{
		Name: agentName + "-agent-loop",
		Steps: []DAGStep{
			{
				Name:     "generate",
				TaskName: "llm-generate",
			},
			{
				Name:      "execute-tools",
				TaskName:  "tool-execute",
				DependsOn: []string{"generate"},
			},
			{
				Name:      "check-done",
				TaskName:  "loop-check",
				DependsOn: []string{"execute-tools"},
			},
		},
	}
}

// AddStep adds a custom step to the DAG.
func (w *DAGWorkflow) AddStep(step DAGStep) {
	w.Steps = append(w.Steps, step)
}

// Validate checks that all dependencies reference existing steps.
func (w *DAGWorkflow) Validate() error {
	names := make(map[string]bool)
	for _, s := range w.Steps {
		names[s.Name] = true
	}
	for _, s := range w.Steps {
		for _, dep := range s.DependsOn {
			if !names[dep] {
				return &validationError{step: s.Name, dep: dep}
			}
		}
	}
	return nil
}

type validationError struct {
	step string
	dep  string
}

func (e *validationError) Error() string {
	return "hatchet: step " + e.step + " depends on unknown step " + e.dep
}
