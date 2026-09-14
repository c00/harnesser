package openrouter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/c00/harnesser/llm"
	"github.com/c00/harnesser/models"
	openrouter "github.com/revrost/go-openrouter"
)

type OpenRouter struct {
	config models.LlmConfig

	ctx    context.Context
	client *openrouter.Client
	logger *slog.Logger
}

var _ llm.LlmProvider = (*OpenRouter)(nil)
var _ llm.LlmStreamingProvider = (*OpenRouter)(nil)

// var _ llm.StructuredOutputProvider = (*OpenRouter)(nil)

func New(ctx context.Context, cfg models.LlmConfig, apiKey string) *OpenRouter {
	return &OpenRouter{
		ctx:    ctx,
		config: cfg,
		client: openrouter.NewClient(apiKey, openrouter.WithXTitle("harnesser")),
		logger: slog.Default().With("package", "openrouter"),
	}
}

func (o *OpenRouter) Config() models.LlmConfig {
	return o.config
}

func convertMessages(messages []models.Message) []openrouter.ChatCompletionMessage {
	orMessages := make([]openrouter.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		orMsg := openrouter.ChatCompletionMessage{
			Role: string(msg.Role),
		}

		var parts []openrouter.ChatMessagePart
		for _, part := range msg.Content {
			var cc *openrouter.CacheControl
			switch part.CacheControl {
			case models.CacheModeShort:
				cc = &openrouter.CacheControl{Type: "ephemeral"}
			case models.CacheModeLong:
				ttl := "1h"
				cc = &openrouter.CacheControl{Type: "ephemeral", TTL: &ttl}
			}

			if part.Type == "text" {
				parts = append(parts, openrouter.ChatMessagePart{
					Type:         openrouter.ChatMessagePartTypeText,
					Text:         part.Text,
					CacheControl: cc,
				})
			} else if part.Type == "image_url" && part.Attachment != nil {
				slog.Debug("image url", "attachment", part.Attachment)
				// Represent attachments as a text placeholder. We do not send
				// image bytes to the LLM — they may contain sensitive financial
				// data (payment screenshots) and would waste tokens
				parts = append(parts, openrouter.ChatMessagePart{
					Type: openrouter.ChatMessagePartTypeText,
					Text: fmt.Sprintf("[attachment: %s]", part.Attachment.Filename),
				})
			}
		}
		orMsg.Content.Multi = parts

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			orToolCalls := make([]openrouter.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				orToolCalls[j] = openrouter.ToolCall{
					ID:   tc.ToolCallID,
					Type: openrouter.ToolTypeFunction,
					Function: openrouter.FunctionCall{
						Name:      tc.Function,
						Arguments: tc.Args,
					},
				}
			}
			orMsg.ToolCalls = orToolCalls
		}

		// Handle tool call ID (for tool role)
		if msg.ToolID != "" {
			orMsg.ToolCallID = msg.ToolID
		}

		orMessages[i] = orMsg
	}
	return orMessages
}

func convertTools(tools []models.Tool) []openrouter.Tool {
	if len(tools) == 0 {
		return nil
	}
	orTools := make([]openrouter.Tool, len(tools))
	for i, t := range tools {
		orTools[i] = openrouter.Tool{
			Type: openrouter.ToolTypeFunction,
			Function: &openrouter.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
	}
	return orTools
}

func (o *OpenRouter) completionRequest(ctx context.Context, messages []models.Message, tools []models.Tool) openrouter.ChatCompletionRequest {
	orMessages := convertMessages(messages)
	if len(orMessages) > 0 {
		last := &orMessages[len(orMessages)-1]
		if len(last.Content.Multi) > 0 {
			o.logger.DebugContext(ctx, "Setting cache control on last message")
			last.Content.Multi[len(last.Content.Multi)-1].CacheControl = &openrouter.CacheControl{Type: "ephemeral"}
		} else {
			o.logger.DebugContext(ctx, "no multi for last message")
		}
	}

	var reasoningEffort *string
	if o.config.Reasoning != "" {
		effort := string(o.config.Reasoning)
		reasoningEffort = &effort
	}

	return openrouter.ChatCompletionRequest{
		Models:    o.config.Models,
		MaxTokens: o.config.MaxOutputTokens,
		Messages:  orMessages,
		Tools:     convertTools(tools),
		Reasoning: &openrouter.ChatCompletionReasoning{
			Effort: reasoningEffort,
		},
	}
}

func (o *OpenRouter) Generate(ctx context.Context, messages []models.Message, tools []models.Tool) (models.Message, error) {
	// fmt.Printf("Body: %v", jsonhelpers.TryParseJson(orMessages))

	start := time.Now()
	o.logger.DebugContext(ctx, "Generate", "messageCount", len(messages), "toolCount", len(tools))
	resp, err := o.client.CreateChatCompletion(
		ctx,
		o.completionRequest(ctx, messages, tools),
	)

	if err != nil {
		return models.Message{}, fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return models.Message{}, errors.New("no choices returned from LLM")
	}

	choice := resp.Choices[0].Message

	// fmt.Printf("Body: %v", jsonhelpers.TryParseJson(choice))

	// Convert back to internal Message model
	var outputTokens, inputTokens, cachedTokens int
	if resp.Usage != nil {
		outputTokens = resp.Usage.CompletionTokens
		inputTokens = resp.Usage.PromptTokens
		cachedTokens = resp.Usage.PromptTokenDetails.CachedTokens
	}
	result := models.Message{
		Role:           models.MessageRole(choice.Role),
		Model:          resp.Model,
		OutputTokens:   outputTokens,
		InputTokens:    inputTokens,
		CachedTokens:   cachedTokens,
		GenerationTime: time.Now().Sub(start),
		FinishReason:   models.FinishReason(resp.Choices[0].FinishReason),
	}

	if choice.Content.Text != "" {
		result.Content = []models.MessagePart{{Type: "text", Text: choice.Content.Text}}
	} else if len(choice.Content.Multi) > 0 {
		for _, p := range choice.Content.Multi {
			part := models.MessagePart{Type: string(p.Type)}
			if p.Type == openrouter.ChatMessagePartTypeText {
				part.Text = p.Text
			} else if p.Type == openrouter.ChatMessagePartTypeImageURL && p.ImageURL != nil {
				// If the LLM generates an image, it returns a URL.
				// In a full implementation, we would download this image,
				// store it in FileStore, create an Attachment record,
				// and set the AttachmentID here.
				// For now, we'll store the URL in the Text field as a placeholder.
				part.Text = p.ImageURL.URL
			}
			result.Content = append(result.Content, part)
		}
	}

	// Handle reasoning details
	if len(choice.ReasoningDetails) > 0 {
		result.Reasoning = convertReasoningDetails(choice.ReasoningDetails)
	}

	if len(choice.ToolCalls) > 0 {
		for _, tc := range choice.ToolCalls {
			result.ToolCalls = append(result.ToolCalls, models.ToolCall{
				ToolCallID: tc.ID,
				Function:   tc.Function.Name,
				Args:       tc.Function.Arguments,
			})
		}
	}

	return result, nil
}

func convertReasoningDetails(details []openrouter.ChatCompletionReasoningDetails) []models.ReasoningDetails {
	result := make([]models.ReasoningDetails, len(details))
	for i, detail := range details {
		result[i] = models.ReasoningDetails{
			ReasoningID: detail.ID,
			Index:       detail.Index,
			Type:        string(detail.Type),
			Text:        detail.Text,
			Summary:     detail.Summary,
			Data:        detail.Data,
			Format:      detail.Format,
		}
	}
	return result
}

func (o *OpenRouter) GenerateStream(ctx context.Context, messages []models.Message, tools []models.Tool, callback llm.StreamDeltaFunc) (models.Message, error) {
	if callback == nil {
		return models.Message{}, errors.New("stream callback is nil")
	}

	request := o.completionRequest(ctx, messages, tools)
	request.Stream = true
	request.StreamOptions = &openrouter.StreamOptions{IncludeUsage: true}

	start := time.Now()
	o.logger.DebugContext(ctx, "GenerateStream", "messageCount", len(messages), "toolCount", len(tools))
	stream, err := o.client.CreateChatCompletionStream(ctx, request)
	if err != nil {
		return models.Message{}, fmt.Errorf("chat completion stream error: %w", err)
	}
	defer stream.Close()

	result := models.Message{Role: models.RoleAssistant}
	var textContent strings.Builder
	reasoningIndexes := make(map[int]int)
	toolCallIndexes := make(map[int]int)
	sawChoice := false
	sawFinish := false

	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			if err := ctx.Err(); err != nil {
				return models.Message{}, fmt.Errorf("chat completion stream error: %w", err)
			}
			break
		}
		if recvErr != nil {
			return models.Message{}, fmt.Errorf("chat completion stream error: %w", recvErr)
		}

		if chunk.Model != "" {
			result.Model = chunk.Model
		}
		if chunk.Usage != nil {
			result.OutputTokens = chunk.Usage.CompletionTokens
			result.InputTokens = chunk.Usage.PromptTokens
			result.CachedTokens = chunk.Usage.PromptTokenDetails.CachedTokens
		}

		delta := models.MessageDelta{}
		if len(chunk.Choices) > 0 {
			sawChoice = true
			choice := chunk.Choices[0]
			if choice.Delta.Role != "" {
				result.Role = models.MessageRole(choice.Delta.Role)
			}

			delta.Text = choice.Delta.Content
			textContent.WriteString(delta.Text)
			delta.Reasoning = convertReasoningDetails(choice.Delta.ReasoningDetails)
			for _, reasoning := range delta.Reasoning {
				position, ok := reasoningIndexes[reasoning.Index]
				if !ok {
					reasoningIndexes[reasoning.Index] = len(result.Reasoning)
					result.Reasoning = append(result.Reasoning, reasoning)
					continue
				}

				assembled := &result.Reasoning[position]
				if reasoning.ReasoningID != "" {
					assembled.ReasoningID = reasoning.ReasoningID
				}
				if reasoning.Type != "" {
					assembled.Type = reasoning.Type
				}
				assembled.Text += reasoning.Text
				assembled.Summary += reasoning.Summary
				assembled.Data += reasoning.Data
				if reasoning.Format != "" {
					assembled.Format = reasoning.Format
				}
			}

			for position, toolCall := range choice.Delta.ToolCalls {
				index := position
				if toolCall.Index != nil {
					index = *toolCall.Index
				}
				delta.ToolCalls = append(delta.ToolCalls, models.ToolCallDelta{
					Index:     index,
					ID:        toolCall.ID,
					Function:  toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				})

				assembledPosition, ok := toolCallIndexes[index]
				if !ok {
					toolCallIndexes[index] = len(result.ToolCalls)
					result.ToolCalls = append(result.ToolCalls, models.ToolCall{
						ToolCallID: toolCall.ID,
						Function:   toolCall.Function.Name,
						Args:       toolCall.Function.Arguments,
					})
					continue
				}

				assembled := &result.ToolCalls[assembledPosition]
				if toolCall.ID != "" {
					assembled.ToolCallID = toolCall.ID
				}
				if toolCall.Function.Name != "" {
					assembled.Function = toolCall.Function.Name
				}
				assembled.Args += toolCall.Function.Arguments
			}

			if choice.FinishReason != "" && choice.FinishReason != openrouter.FinishReasonNull {
				result.FinishReason = models.FinishReason(choice.FinishReason)
				sawFinish = true
			}
		}

		callback(delta)
	}

	if !sawChoice {
		return models.Message{}, errors.New("no choices returned from LLM stream")
	}
	if !sawFinish {
		return models.Message{}, errors.New("LLM stream ended before a finish reason was returned")
	}
	if textContent.Len() > 0 {
		result.Content = []models.MessagePart{{Type: models.PartText, Text: textContent.String()}}
	}
	result.GenerationTime = time.Since(start)

	return result, nil
}
