package models

import (
	"strings"
	"time"
)

// MessageRole defines who sent the message.
type MessageRole string
type FinishReason string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"

	// MessagePart types
	PartText  = "text"
	PartImage = "image_url"
	PartAudio = "audio"

	// Finish reasons
	FinishReasonStop      FinishReason = "stop"
	FinishReasonLength    FinishReason = "length"
	FinishReasonToolCalls FinishReason = "tool_calls"
)

// Message stores the conversation context for the LLM and the user.
type Message struct {
	Role           MessageRole    `json:"role"`
	Content        MessageParts   `json:"content"`
	ToolCalls      []ToolCall     `json:"tool_calls,omitempty"`
	ToolID         string         `json:"tool_call_id,omitempty"` // For RoleTool
	ToolName       string         `json:"tool_name"`              // For RoleTool
	CreatedAt      time.Time      `json:"created_at"`
	Reasoning      ReasoningParts `json:"reasoning_details"`
	Model          string         `json:"model"`
	GenerationTime time.Duration  `json:"generation_time"`
	OutputTokens   int            `json:"output_tokens"`
	InputTokens    int            `json:"input_tokens"`
	CachedTokens   int            `json:"cached_tokens"`

	// Not persisted in database
	FinishReason FinishReason
}

type Messages []Message
type MessageParts []MessagePart
type ReasoningParts []ReasoningDetails

// MessagePart represents a piece of content in a message.
type MessagePart struct {
	Type         string      `json:"type"` // "text", "image_url", "audio"
	Text         string      `json:"text,omitempty"`
	Attachment   *Attachment `json:"attachment,omitempty"` // Reference to Attachment table
	CacheControl CacheMode   `json:"-"`                    // In-memory only; not persisted
}

type ReasoningDetails struct {
	// The ID given by the LLM
	ReasoningID string `json:"reasoning_id,omitempty"`
	Index       int    `json:"index"`
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Data        string `json:"data,omitempty"`
	Format      string `json:"format,omitempty"`
}

func NewMessage(role MessageRole, text string) Message {
	return Message{
		Role: role,
		Content: []MessagePart{
			{
				Type: "text",
				Text: text,
			},
		},
		CreatedAt: time.Now(),
	}
}

func NewUserTextMessage(text string) Message {
	return NewMessage(RoleUser, text)
}

func NewToolResultMessage(toolId, text string) Message {
	msg := NewMessage(RoleTool, text)
	msg.ToolID = toolId
	return msg
}

func NewToolErrorMessage(toolId, text string) Message {
	msg := NewMessage(RoleTool, text)
	msg.ToolID = toolId
	return msg
}

func NewAssistantMessage(text string) Message {
	return NewMessage(RoleAssistant, text)
}

func NewAssistantErrorMessage(err error) Message {
	if err == nil {
		return NewMessage(RoleAssistant, "Something went wrong")
	}

	return NewMessage(RoleAssistant, "An error occured: "+err.Error())
}

func NewSystemMessage(text string) Message {
	return NewMessage(RoleSystem, text)
}

func (m MessageParts) String() string {
	var result string
	for i, part := range m {
		if i > 0 {
			result += "\n"
		}
		switch part.Type {
		case "text":
			result += part.Text
		default:
			result += "[" + part.Type + "]"
		}
	}
	return result
}

func (m ReasoningParts) String() string {
	var res string
	for _, r := range m {
		if r.Type == "reasoning.encrypted" {
			res += "\n  - Reasoning encrypted"
		} else if r.Text != "" {
			res += "\n  - Reasoning: " + r.Text
		} else if r.Summary != "" {
			res += "\n  - Reasoning Summary: " + r.Summary
		}
	}
	return res
}

func (ms Messages) String() string {
	parts := make([]string, 0, len(ms))
	for _, m := range ms {
		parts = append(parts, m.String())
	}
	return strings.Join(parts, "\n")
}

func (m Message) String() string {
	res := m.CreatedAt.Format("2006-01-02 15:04:05") + " [" + string(m.Role) + "]: " + m.Content.String()

	if len(m.Reasoning) > 0 {
		res += ReasoningParts(m.Reasoning).String()
	}

	if len(m.ToolCalls) > 0 {
		for _, tc := range m.ToolCalls {
			res += "\n  - Tool Call: " + tc.String()
		}
	}

	if m.ToolID != "" {
		res += "\n  - Tool ID: " + m.ToolID
	}

	return res
}

func (m Message) AddText(text string) Message {
	if m.Content == nil {
		m.Content = MessageParts{}
	}
	m.Content = append(m.Content, MessagePart{Type: "text", Text: text})

	return m
}
