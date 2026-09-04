// package mockllm exists for automated tests with an llm provider
package mockllm

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/c00/harnesser/llm"
	"github.com/c00/harnesser/models"
)

var _ llm.LlmProvider = (*MockLLM)(nil)
var _ llm.StructuredOutputProvider = (*MockLLM)(nil)

// MockLLM is a fake LLM that will return responses based on predefined keywords or phrases (case insensitive)
// The response corresponding to the key that appears earliest in the prompt is returned.
type MockLLM struct {
	Response            map[string]models.Message
	StructuredResponses map[string]models.SchemaMarshaller
	logger              *slog.Logger
}

func New() *MockLLM {
	return &MockLLM{
		Response:            make(map[string]models.Message),
		StructuredResponses: make(map[string]models.SchemaMarshaller),
		logger:              slog.Default().With("package", "mockllm"),
	}
}

func (m *MockLLM) Config() models.LlmConfig {
	return models.LlmConfig{
		Name:   "Mock LLM",
		Models: []string{"Mock"},
	}
}

func (m *MockLLM) AddTextResponse(phrase, responseText string) {
	if m.Response == nil {
		m.Response = make(map[string]models.Message)
	}
	m.Response[phrase] = models.Message{
		Role: models.RoleAssistant,
		Content: []models.MessagePart{
			{
				Type: "text",
				Text: responseText,
			},
		},
	}
}

func (m *MockLLM) AddEmptyResponse(phrase string, finishReason models.FinishReason) {
	if m.Response == nil {
		m.Response = make(map[string]models.Message)
	}
	m.Response[phrase] = models.Message{
		Role:         models.RoleAssistant,
		ToolCalls:    []models.ToolCall{},
		Content:      models.MessageParts{},
		FinishReason: finishReason,
	}
}

func (m *MockLLM) AddToolcallResponse(phrase, toolName, args, responseText string) {
	if m.Response == nil {
		m.Response = make(map[string]models.Message)
	}

	msg := models.Message{
		Role: models.RoleAssistant,
		ToolCalls: []models.ToolCall{
			{
				ToolCallID: "call_" + toolName,
				Function:   toolName,
				Args:       args,
			},
		},
	}

	if responseText != "" {
		msg.Content = []models.MessagePart{
			{
				Type: "text",
				Text: responseText,
			},
		}
	}

	m.Response[phrase] = msg
}

func (m *MockLLM) GenerateStructuredOutput(ctx context.Context, messages []models.Message, tools []models.Tool, output models.SchemaMarshaller) (models.LlmUsage, error) {
	if len(messages) == 0 {
		return models.LlmUsage{}, errors.New("no messages")
	}

	// Get the last user message to search for keywords/phrases
	fullText := messages[len(messages)-1].Content.String()

	if m.logger != nil {
		m.logger.DebugContext(ctx, "Generating mock structured response", "messageCount", len(messages), "incoming", fullText)
	}

	lowerText := strings.ToLower(fullText)
	var bestMatch models.SchemaMarshaller
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
		return models.LlmUsage{}, nil
	}

	return models.LlmUsage{}, errors.New("no response defined")
}

func (m *MockLLM) AddStructuredResponse(phrase string, response models.SchemaMarshaller) {
	m.StructuredResponses[phrase] = response
}

func (m *MockLLM) Generate(ctx context.Context, messages []models.Message, tools []models.Tool) (models.Message, error) {
	if len(messages) == 0 {
		return models.Message{}, errors.New("no messages")
	}

	// Get the last user message to search for keywords/phrases
	fullText := messages[len(messages)-1].Content.String()

	if m.logger != nil {
		m.logger.DebugContext(ctx, "Generating mock LLM response", "messageCount", len(messages), "incoming", fullText)
	}

	lowerText := strings.ToLower(fullText)
	var bestMatch *models.Message
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
	return models.Message{
		Role:    models.RoleAssistant,
		Content: []models.MessagePart{{Type: "text", Text: "I don't have a response to that."}},
		Model:   m.Config().Models[0],
	}, nil
}
