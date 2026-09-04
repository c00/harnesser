package openrouter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

func (o *OpenRouter) Generate(ctx context.Context, messages []models.Message, tools []models.Tool) (models.Message, error) {
	orMessages := convertMessages(messages)
	orTools := convertTools(tools)

	if len(orMessages) > 0 {
		last := &orMessages[len(orMessages)-1]
		if len(last.Content.Multi) > 0 {
			o.logger.DebugContext(ctx, "Setting cache control on last message")
			last.Content.Multi[len(last.Content.Multi)-1].CacheControl = &openrouter.CacheControl{Type: "ephemeral"}
		} else {
			o.logger.DebugContext(ctx, "no multi for last message")
		}
	}

	// fmt.Printf("Body: %v", jsonhelpers.TryParseJson(orMessages))

	start := time.Now()
	o.logger.DebugContext(ctx, "Generate", "messageCount", len(messages), "toolCount", len(tools))
	resp, err := o.tryGenerate(ctx, orMessages, orTools, nil)

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
		result.Reasoning = make([]models.ReasoningDetails, len(choice.ReasoningDetails))
		for i, r := range choice.ReasoningDetails {
			result.Reasoning[i] = models.ReasoningDetails{
				ReasoningID: r.ID,
				Index:       r.Index,
				Type:        string(r.Type),
				Text:        r.Text,
				Summary:     r.Summary,
				Data:        r.Data,
				Format:      r.Format,
			}
		}
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

// tryGenerate Will try to generate a response, and retry a few times n failure. It will also switch out models if possible.
func (o *OpenRouter) tryGenerate(ctx context.Context, orMessages []openrouter.ChatCompletionMessage, orTools []openrouter.Tool, responseFormat *openrouter.ChatCompletionResponseFormat) (openrouter.ChatCompletionResponse, error) {
	var resp openrouter.ChatCompletionResponse
	var err error
	try := 0
	maxTries := 1
	for {
		var reasoningEffort *string
		if o.config.Reasoning != "" {
			effort := string(o.config.Reasoning)
			reasoningEffort = &effort
		}
		o.logger.DebugContext(ctx, "Creating Chat Completion", "messageCount", len(orMessages), "toolCount", len(orTools))
		resp, err = o.client.CreateChatCompletion(
			ctx,
			openrouter.ChatCompletionRequest{
				// Model:     model,
				Models:    o.config.Models,
				MaxTokens: o.config.MaxOutputTokens,
				Messages:  orMessages,
				Tools:     orTools,
				Reasoning: &openrouter.ChatCompletionReasoning{
					Effort: reasoningEffort,
				},
				ResponseFormat: responseFormat,
			},
		)

		if err == nil && len(resp.Choices) == 0 {
			err = errors.New("no choices from LLM")
		}
		if err == nil {
			return resp, nil
		}

		// TODO I don't think this is the right type.
		var reqErr *openrouter.RequestError
		if errors.As(err, &reqErr) {
			switch reqErr.HTTPStatusCode {
			case 408, 500, 502, 524:
				// These are retryable, continue the loop
				o.logger.WarnContext(ctx, "Retryable HTTP error", "status", reqErr.HTTPStatusCode)
			default:
				return resp, fmt.Errorf("openrouter request error: %w", err)
			}
		}

		try++
		if try >= maxTries {
			return resp, fmt.Errorf("too many retries: %w", err)
		}

		o.logger.WarnContext(ctx, "LLM call failed, will retry...", "error", err.Error())

		// Backoff a little bit before retrying
		time.Sleep(time.Duration(try) * time.Second)
	}
}
