package agui

import "time"

// EventType identifies an AG-UI protocol event.
type EventType string

const (
	EventTypeRunStarted           EventType = "run_started"
	EventTypeRunFinished          EventType = "run_finished"
	EventTypeTextMessageStart     EventType = "text_message_start"
	EventTypeTextMessageContent   EventType = "text_message_content"
	EventTypeTextMessageEnd       EventType = "text_message_end"
	EventTypeToolCallStart          EventType = "tool_call_start"
	EventTypeToolCallArgs           EventType = "tool_call_args"
	EventTypeToolCallEnd            EventType = "tool_call_end"
	EventTypeStateSnapshot          EventType = "state_snapshot"
	EventTypeStateDelta             EventType = "state_delta"
	EventTypeMessagesSnapshot       EventType = "messages_snapshot"
	EventTypeStepStarted            EventType = "step_started"
	EventTypeStepFinished           EventType = "step_finished"
)

// BaseEvent is the common envelope for all AG-UI events.
type BaseEvent struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// RunStartedEvent signals the beginning of an agent run.
type RunStartedEvent struct {
	BaseEvent
	RunID string `json:"run_id"`
}

// RunFinishedEvent signals the completion of an agent run.
type RunFinishedEvent struct {
	BaseEvent
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// TextMessageStartEvent signals the start of a text message stream.
type TextMessageStartEvent struct {
	BaseEvent
	MessageID string `json:"message_id"`
	Role      string `json:"role"`
}

// TextMessageContentEvent carries a chunk of text message content.
type TextMessageContentEvent struct {
	BaseEvent
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
}

// TextMessageEndEvent signals the end of a text message stream.
type TextMessageEndEvent struct {
	BaseEvent
	MessageID string `json:"message_id"`
}

// ToolCallStartEvent signals the beginning of a tool invocation.
type ToolCallStartEvent struct {
	BaseEvent
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
}

// ToolCallArgsEvent carries incremental tool call arguments.
type ToolCallArgsEvent struct {
	BaseEvent
	ToolCallID string `json:"tool_call_id"`
	Args       string `json:"args"`
}

// ToolCallEndEvent signals the completion of a tool invocation.
type ToolCallEndEvent struct {
	BaseEvent
	ToolCallID string `json:"tool_call_id"`
	Result     string `json:"result,omitempty"`
}

// StateSnapshotEvent delivers a full state snapshot.
type StateSnapshotEvent struct {
	BaseEvent
	State map[string]any `json:"state"`
}

// StateDeltaEvent delivers an incremental state update.
type StateDeltaEvent struct {
	BaseEvent
	Delta map[string]any `json:"delta"`
}

// MessagesSnapshotEvent delivers a snapshot of the conversation history.
type MessagesSnapshotEvent struct {
	BaseEvent
	Messages []map[string]any `json:"messages"`
}

// StepStartedEvent signals the beginning of an agent step.
type StepStartedEvent struct {
	BaseEvent
	StepID string `json:"step_id"`
	Name   string `json:"name"`
}

// StepFinishedEvent signals the completion of an agent step.
type StepFinishedEvent struct {
	BaseEvent
	StepID string `json:"step_id"`
	Status string `json:"status"`
}

// EventStream is a channel of AG-UI protocol events.
type EventStream chan BaseEvent
