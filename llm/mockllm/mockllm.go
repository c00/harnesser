// package mockllm exists for automated tests with an llm provider
package mockllm

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/c00/harnesser/llm"
	"github.com/c00/harnesser/types"
)

var _ llm.LlmProvider = (*MockLLM)(nil)
var _ llm.LlmStreamingProvider = (*MockLLM)(nil)

// var _ llm.StructuredOutputProvider = (*MockLLM)(nil)

// MockLLM is a fake LLM that will return responses based on predefined keywords or phrases (case insensitive)
// The response corresponding to the key that appears earliest in the prompt is returned.
type MockLLM struct {
	Response            map[string]types.Message
	StructuredResponses map[string]types.SchemaMarshaller
	logger              *slog.Logger
}

func New() *MockLLM {
	return &MockLLM{
		Response:            make(map[string]types.Message),
		StructuredResponses: make(map[string]types.SchemaMarshaller),
		logger:              slog.Default().With("package", "mockllm"),
	}
}

func (m *MockLLM) Config() types.LlmConfig {
	return types.LlmConfig{
		Name:   "Mock LLM",
		Models: []string{"Mock"},
	}
}

func (m *MockLLM) AddTextResponse(phrase, responseText string) {
	if m.Response == nil {
		m.Response = make(map[string]types.Message)
	}
	m.Response[phrase] = types.Message{
		Role: types.RoleAssistant,
		Content: []types.MessagePart{
			{
				Type: "text",
				Text: responseText,
			},
		},
	}
}

func (m *MockLLM) AddEmptyResponse(phrase string, finishReason types.FinishReason) {
	if m.Response == nil {
		m.Response = make(map[string]types.Message)
	}
	m.Response[phrase] = types.Message{
		Role:         types.RoleAssistant,
		ToolCalls:    []types.ToolCall{},
		Content:      types.MessageParts{},
		FinishReason: finishReason,
	}
}

func (m *MockLLM) AddToolcallResponse(phrase, toolName, args, responseText string) {
	if m.Response == nil {
		m.Response = make(map[string]types.Message)
	}

	msg := types.Message{
		Role: types.RoleAssistant,
		ToolCalls: []types.ToolCall{
			{
				ToolCallID: "call_" + toolName,
				Function:   toolName,
				Args:       args,
			},
		},
	}

	if responseText != "" {
		msg.Content = []types.MessagePart{
			{
				Type: "text",
				Text: responseText,
			},
		}
	}

	m.Response[phrase] = msg
}

func (m *MockLLM) GenerateStructuredOutput(ctx context.Context, messages []types.Message, tools []types.Tool, output types.SchemaMarshaller) (types.LlmUsage, error) {
	if len(messages) == 0 {
		return types.LlmUsage{}, errors.New("no messages")
	}

	// Get the last user message to search for keywords/phrases
	fullText := messages[len(messages)-1].Content.String()

	if m.logger != nil {
		m.logger.DebugContext(ctx, "Generating mock structured response", "messageCount", len(messages), "incoming", fullText)
	}

	lowerText := strings.ToLower(fullText)
	var bestMatch types.SchemaMarshaller
	firstIndex := -1

	for phrase, resp := range m.StructuredResponses {
		idx := strings.Index(lowerText, strings.ToLower(phrase))
		if idx != -1 {
			// If this phrase appears earlier than any previous match, select it
			if firstIndex == -1 || idx < firstIndex {
				firstIndex = idx
				val := resp
				bestMatch = val
			}
		}
	}

	if bestMatch != nil {
		output = bestMatch
		return types.LlmUsage{}, nil
	}

	return types.LlmUsage{}, errors.New("no response defined")
}

func (m *MockLLM) AddStructuredResponse(phrase string, response types.SchemaMarshaller) {
	m.StructuredResponses[phrase] = response
}

func (m *MockLLM) Generate(ctx context.Context, messages []types.Message, tools []types.Tool) (types.Message, error) {
	if len(messages) == 0 {
		return types.Message{}, errors.New("no messages")
	}

	// Get the last user message to search for keywords/phrases
	fullText := messages[len(messages)-1].Content.String()

	if m.logger != nil {
		m.logger.DebugContext(ctx, "Generating mock LLM response", "messageCount", len(messages), "incoming", fullText)
	}

	lowerText := strings.ToLower(fullText)
	var bestMatch *types.Message
	firstIndex := -1

	for phrase, resp := range m.Response {
		idx := strings.Index(lowerText, strings.ToLower(phrase))
		if idx != -1 {
			// If this phrase appears earlier than any previous match, select it
			if firstIndex == -1 || idx < firstIndex {
				firstIndex = idx
				val := resp
				bestMatch = &val
			}
		}
	}

	if bestMatch != nil {
		return *bestMatch, nil
	}

	// Default response if no keyword matches
	return types.Message{
		Role:    types.RoleAssistant,
		Content: []types.MessagePart{{Type: "text", Text: "I don't have a response to that."}},
		Model:   m.Config().Models[0],
	}, nil
}

func (m *MockLLM) GenerateStream(ctx context.Context, messages []types.Message, tools []types.Tool, callback llm.StreamDeltaFunc) (types.Message, error) {
	if callback == nil {
		return types.Message{}, errors.New("stream callback is nil")
	}

	response, err := m.Generate(ctx, messages, tools)
	if err != nil {
		return types.Message{}, err
	}

	delta := types.MessageDelta{Reasoning: response.Reasoning}
	for _, part := range response.Content {
		if part.Type == types.PartText {
			delta.Text += part.Text
		}
	}
	for index, toolCall := range response.ToolCalls {
		delta.ToolCalls = append(delta.ToolCalls, types.ToolCallDelta{
			Index:     index,
			ID:        toolCall.ToolCallID,
			Function:  toolCall.Function,
			Arguments: toolCall.Args,
		})
	}

	callback(delta)
	return response, nil
}
