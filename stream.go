package graft

import "time"

type EventType string

const (
	EventTextDelta      EventType = "text_delta"
	EventToolCallStart  EventType = "tool_call_start"
	EventToolCallDelta  EventType = "tool_call_delta"
	EventToolCallDone   EventType = "tool_call_done"
	EventToolResultDone EventType = "tool_result_done"
	EventMessageDone    EventType = "message_done"
	EventHandoff        EventType = "handoff"
	EventError          EventType = "error"
	EventDone           EventType = "done"
)

type StreamEvent struct {
	Type      EventType `json:"type"`
	Data      any       `json:"data"`
	AgentID   string    `json:"agent_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
