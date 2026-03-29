package trigger

// Waitpoint represents a pause point in task execution.
// Trigger.dev's Waitpoints allow tasks to pause indefinitely and
// resume when external data arrives, at zero compute cost.
type Waitpoint struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Token       string `json:"token"`
	Type        WaitpointType `json:"type"`
}

// WaitpointType describes the kind of waitpoint.
type WaitpointType string

const (
	WaitpointApproval WaitpointType = "approval"
	WaitpointData     WaitpointType = "data"
)

// WaitpointHandler handles approval workflows.
type WaitpointHandler interface {
	OnWaitpoint(wp Waitpoint) error
}

// NewApprovalWaitpoint creates a waitpoint that pauses for human approval.
func NewApprovalWaitpoint(description string) *Waitpoint {
	return &Waitpoint{
		Description: description,
		Type:        WaitpointApproval,
	}
}

// ApprovalData is the data sent to complete an approval waitpoint.
type ApprovalData struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}
