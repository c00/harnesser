package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMessageParts_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		parts    MessageParts
		expected string
	}{
		{
			name: "single text part",
			parts: MessageParts{
				{Type: "text", Text: "Hello world"},
			},
			expected: "Hello world",
		},
		{
			name: "multiple text parts",
			parts: MessageParts{
				{Type: "text", Text: "Hello"},
				{Type: "text", Text: "world"},
			},
			expected: "Hello\nworld",
		},
		{
			name: "mixed text and attachment parts",
			parts: MessageParts{
				{Type: "text", Text: "Check this out:"},
				{Type: "image_url", Attachment: &Attachment{}},
				{Type: "audio", Attachment: &Attachment{}},
			},
			expected: "Check this out:\n[image_url]\n[audio]",
		},
		{
			name:     "empty parts",
			parts:    MessageParts{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.parts.String())
		})
	}
}

func TestMessage_String(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name: "simple user message",
			message: Message{
				Role: RoleUser,
				Content: MessageParts{
					{Type: "text", Text: "Hello"},
				},
				CreatedAt: fixedTime,
			},
			expected: "2023-10-27 10:00:00 [user]: Hello",
		},
		{
			name: "assistant message with tool calls",
			message: Message{
				Role: RoleAssistant,
				Content: MessageParts{
					{Type: "text", Text: "Let me check that."},
				},
				ToolCalls: []ToolCall{
					{ToolCallID: "call_1", Function: "get_order", Args: `{"id": 1}`},
				},
				CreatedAt: fixedTime,
			},
			expected: "2023-10-27 10:00:00 [assistant]: Let me check that.\n  - Tool Call: get_order (call_1): with arguments",
		},
		{
			name: "tool response message",
			message: Message{
				Role:   RoleTool,
				ToolID: "call_1",
				Content: MessageParts{
					{Type: "text", Text: "Order found"},
				},
				CreatedAt: fixedTime,
			},
			expected: "2023-10-27 10:00:00 [tool]: Order found\n  - Tool ID: call_1",
		},
		{
			name: "assistant message with reasoning",
			message: Message{
				Role: RoleAssistant,
				Content: MessageParts{
					{Type: "text", Text: "The answer is 42."},
				},
				Reasoning: []ReasoningDetails{
					{Type: "text", Text: "Calculating the ultimate answer..."},
				},
				CreatedAt: fixedTime,
			},
			expected: "2023-10-27 10:00:00 [assistant]: The answer is 42.\n  - Reasoning: Calculating the ultimate answer...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.message.String())
		})
	}
}
