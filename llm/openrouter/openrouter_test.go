//go:build integration

package openrouter

import (
	"os"
	"reflect"
	"testing"

	"github.com/c00/harnesser/llm"
	"github.com/c00/harnesser/llm/llmtestsuite"
	"github.com/c00/harnesser/models"
	openrouterlib "github.com/revrost/go-openrouter"
)

func ptr(s string) *string { return &s }

func TestConvertMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []models.Message
		want  []openrouterlib.ChatCompletionMessage
	}{
		{
			name: "simple text",
			input: []models.Message{
				{Role: models.RoleUser, Content: []models.MessagePart{{Type: "text", Text: "hello"}}},
			},
			want: []openrouterlib.ChatCompletionMessage{
				{Role: "user", Content: openrouterlib.Content{Multi: []openrouterlib.ChatMessagePart{
					{Type: openrouterlib.ChatMessagePartTypeText, Text: "hello"},
				}}},
			},
		},
		{
			name: "attachment only becomes text placeholder",
			input: []models.Message{
				{
					Role: models.RoleUser,
					Content: []models.MessagePart{
						{Type: "image_url", Attachment: &models.Attachment{Filename: "receipt.png"}},
					},
				},
			},
			want: []openrouterlib.ChatCompletionMessage{
				{Role: "user", Content: openrouterlib.Content{Multi: []openrouterlib.ChatMessagePart{
					{Type: openrouterlib.ChatMessagePartTypeText, Text: "[attachment: receipt.png]"},
				}}},
			},
		},
		{
			name: "text and attachment",
			input: []models.Message{
				{
					Role: models.RoleUser,
					Content: []models.MessagePart{
						{Type: "text", Text: "see attached"},
						{Type: "image_url", Attachment: &models.Attachment{Filename: "invoice.jpg"}},
					},
				},
			},
			want: []openrouterlib.ChatCompletionMessage{
				{Role: "user", Content: openrouterlib.Content{Multi: []openrouterlib.ChatMessagePart{
					{Type: openrouterlib.ChatMessagePartTypeText, Text: "see attached"},
					{Type: openrouterlib.ChatMessagePartTypeText, Text: "[attachment: invoice.jpg]"},
				}}},
			},
		},
		{
			name: "cache control short",
			input: []models.Message{
				{Role: models.RoleSystem, Content: []models.MessagePart{
					{Type: "text", Text: "sys", CacheControl: models.CacheModeShort},
				}},
			},
			want: []openrouterlib.ChatCompletionMessage{
				{Role: "system", Content: openrouterlib.Content{Multi: []openrouterlib.ChatMessagePart{
					{Type: openrouterlib.ChatMessagePartTypeText, Text: "sys", CacheControl: &openrouterlib.CacheControl{Type: "ephemeral"}},
				}}},
			},
		},
		{
			name: "cache control long",
			input: []models.Message{
				{Role: models.RoleSystem, Content: []models.MessagePart{
					{Type: "text", Text: "sys", CacheControl: models.CacheModeLong},
				}},
			},
			want: []openrouterlib.ChatCompletionMessage{
				{Role: "system", Content: openrouterlib.Content{Multi: []openrouterlib.ChatMessagePart{
					{Type: openrouterlib.ChatMessagePartTypeText, Text: "sys", CacheControl: &openrouterlib.CacheControl{Type: "ephemeral", TTL: ptr("1h")}},
				}}},
			},
		},
		{
			name: "tool calls",
			input: []models.Message{
				{
					Role: models.RoleAssistant,
					ToolCalls: []models.ToolCall{
						{ToolCallID: "call_1", Function: "get_order", Args: `{"id":1}`},
					},
				},
			},
			want: []openrouterlib.ChatCompletionMessage{
				{
					Role:    "assistant",
					Content: openrouterlib.Content{Multi: nil},
					ToolCalls: []openrouterlib.ToolCall{
						{ID: "call_1", Type: openrouterlib.ToolTypeFunction, Function: openrouterlib.FunctionCall{Name: "get_order", Arguments: `{"id":1}`}},
					},
				},
			},
		},
		{
			name: "tool role",
			input: []models.Message{
				{Role: models.RoleTool, ToolID: "call_1", Content: []models.MessagePart{{Type: "text", Text: "result"}}},
			},
			want: []openrouterlib.ChatCompletionMessage{
				{Role: "tool", ToolCallID: "call_1", Content: openrouterlib.Content{Multi: []openrouterlib.ChatMessagePart{
					{Type: openrouterlib.ChatMessagePartTypeText, Text: "result"},
				}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := convertMessages(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertMessages() =\n  %+v\nwant\n  %+v", got, tt.want)
			}
		})
	}
}

func TestOpenRouter_LlmTestSuite(t *testing.T) {
	t.Parallel()

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("No OPENROUTER_API_KEY set. Skipping openrouter llm test")
	}

	cfg := models.LlmConfig{
		Models:          []string{"openai/gpt-3.5-turbo-16k"},
		Name:            "openrouter",
		MaxOutputTokens: 25,
	}

	llmtestsuite.RunLlmTestSuite(t, func() llm.LlmProvider {
		llm := New(t.Context(), cfg, apiKey)
		return llm
	})
}
