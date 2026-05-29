package agui

import "testing"

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name  string
		value EventType
		want  string
	}{
		{"RunStarted", EventTypeRunStarted, "run_started"},
		{"RunFinished", EventTypeRunFinished, "run_finished"},
		{"TextMessageStart", EventTypeTextMessageStart, "text_message_start"},
		{"TextMessageContent", EventTypeTextMessageContent, "text_message_content"},
		{"TextMessageEnd", EventTypeTextMessageEnd, "text_message_end"},
		{"ToolCallStart", EventTypeToolCallStart, "tool_call_start"},
		{"ToolCallArgs", EventTypeToolCallArgs, "tool_call_args"},
		{"ToolCallEnd", EventTypeToolCallEnd, "tool_call_end"},
		{"StateSnapshot", EventTypeStateSnapshot, "state_snapshot"},
		{"StateDelta", EventTypeStateDelta, "state_delta"},
		{"MessagesSnapshot", EventTypeMessagesSnapshot, "messages_snapshot"},
		{"StepStarted", EventTypeStepStarted, "step_started"},
		{"StepFinished", EventTypeStepFinished, "step_finished"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.want {
				t.Errorf("got %q, want %q", tt.value, tt.want)
			}
		})
	}
}
