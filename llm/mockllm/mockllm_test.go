package mockllm_test

import (
	"testing"

	"github.com/c00/harnesser/llm"
	"github.com/c00/harnesser/llm/llmtestsuite"
	"github.com/c00/harnesser/llm/mockllm"
	"github.com/c00/harnesser/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockLLM_LlmTestSuite(t *testing.T) {
	t.Parallel()
	llmtestsuite.RunLlmTestSuite(t, func() llm.LlmProvider {
		mock := &mockllm.MockLLM{}
		mock.AddTextResponse("hi", "hi")
		mock.AddToolcallResponse("weather", "get_weather", `{"city":"Amsterdam"}`, "Let me get that for you...")
		return mock
	})
}

func TestMockLLM_AddTextResponse(t *testing.T) {
	t.Parallel()
	mock := &mockllm.MockLLM{}
	mock.AddTextResponse("test phrase", "test response")

	require.NotNil(t, mock.Response)
	resp, ok := mock.Response["test phrase"]
	assert.True(t, ok)
	assert.Equal(t, models.RoleAssistant, resp.Role)
	assert.Equal(t, "test response", resp.Content[0].Text)
}

func TestMockLLM_AddToolcallResponse(t *testing.T) {
	t.Parallel()
	mock := &mockllm.MockLLM{}
	mock.AddToolcallResponse("trigger", "get_weather", `{"location": "London"}`, "Checking weather...")

	require.NotNil(t, mock.Response)
	resp, ok := mock.Response["trigger"]
	assert.True(t, ok)
	assert.Equal(t, models.RoleAssistant, resp.Role)
	assert.Equal(t, "Checking weather...", resp.Content[0].Text)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "get_weather", resp.ToolCalls[0].Function)
	assert.Equal(t, `{"location": "London"}`, resp.ToolCalls[0].Args)
}

func TestMockLLM_Generate(t *testing.T) {
	t.Parallel()
	responses := map[string]models.Message{
		"hello": {
			Role:    models.RoleAssistant,
			Content: []models.MessagePart{{Type: "text", Text: "Hi there!"}},
		},
		"order pizza": {
			Role:    models.RoleAssistant,
			Content: []models.MessagePart{{Type: "text", Text: "Ordering pizza now."}},
		},
	}

	mock := &mockllm.MockLLM{Response: responses}

	tests := []struct {
		name          string
		messages      []models.Message
		expectedText  string
		expectedError bool
	}{
		{
			name: "Match first keyword",
			messages: []models.Message{
				{Role: models.RoleUser, Content: []models.MessagePart{{Type: "text", Text: "Hello, I'd like to help."}}},
			},
			expectedText: "Hi there!",
		},
		{
			name: "Match based on earliest appearance",
			messages: []models.Message{
				{Role: models.RoleUser, Content: []models.MessagePart{{Type: "text", Text: "Please order pizza and say hello."}}},
			},
			expectedText: "Ordering pizza now.",
		},
		{
			name: "No match returns default",
			messages: []models.Message{
				{Role: models.RoleUser, Content: []models.MessagePart{{Type: "text", Text: "What is the weather?"}}},
			},
			expectedText: "I don't have a response to that.",
		},
		{
			name:          "Empty messages returns error",
			messages:      []models.Message{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := mock.Generate(t.Context(), tt.messages, []models.Tool{})

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, models.RoleAssistant, resp.Role)
			assert.Equal(t, tt.expectedText, resp.Content[0].Text)
		})
	}
}

func TestMockLLM_GenerateStream(t *testing.T) {
	t.Parallel()

	mock := &mockllm.MockLLM{}
	mock.AddToolcallResponse("weather", "get_weather", `{"city":"Amsterdam"}`, "Let me check.")
	mock.Response["weather"] = models.Message{
		Role:      mock.Response["weather"].Role,
		Content:   mock.Response["weather"].Content,
		ToolCalls: mock.Response["weather"].ToolCalls,
		Reasoning: []models.ReasoningDetails{
			{Index: 0, Type: "reasoning.text", Text: "Need the weather."},
		},
	}

	var deltas []models.MessageDelta
	result, err := mock.GenerateStream(t.Context(), []models.Message{
		models.NewUserTextMessage("What is the weather?"),
	}, nil, func(delta models.MessageDelta) {
		deltas = append(deltas, delta)
	})

	require.NoError(t, err)
	assert.Equal(t, mock.Response["weather"], result)
	require.Len(t, deltas, 1)
	assert.Equal(t, "Let me check.", deltas[0].Text)
	assert.Equal(t, []models.ReasoningDetails(result.Reasoning), deltas[0].Reasoning)
	require.Len(t, deltas[0].ToolCalls, 1)
	assert.Equal(t, 0, deltas[0].ToolCalls[0].Index)
	assert.Equal(t, "call_get_weather", deltas[0].ToolCalls[0].ID)
	assert.Equal(t, "get_weather", deltas[0].ToolCalls[0].Function)
	assert.Equal(t, `{"city":"Amsterdam"}`, deltas[0].ToolCalls[0].Arguments)
}

func TestMockLLM_GenerateStreamRejectsNilCallback(t *testing.T) {
	t.Parallel()

	mock := mockllm.New()
	_, err := mock.GenerateStream(t.Context(), nil, nil, nil)

	require.EqualError(t, err, "stream callback is nil")
}
