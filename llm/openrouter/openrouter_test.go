package openrouter

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/c00/harnesser/models"
	openrouterlib "github.com/revrost/go-openrouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateDoesNotRetry(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, `{"error":{"message":"temporary failure"}}`, http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	clientConfig := openrouterlib.DefaultConfig("test-api-key")
	clientConfig.BaseURL = server.URL
	provider := New(t.Context(), models.LlmConfig{Models: []string{"test/model"}}, "test-api-key")
	provider.client = openrouterlib.NewClientWithConfig(*clientConfig)

	_, err := provider.Generate(t.Context(), []models.Message{
		{Role: models.RoleUser, Content: []models.MessagePart{{Type: "text", Text: "hello"}}},
	}, nil)

	require.Error(t, err)
	assert.Equal(t, int32(1), requestCount.Load())
}
